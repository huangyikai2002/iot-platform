package mqtt

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"iot-platform/internal/ai"
	"iot-platform/internal/config"
	"iot-platform/internal/domain"
	mysqlstore "iot-platform/internal/store/mysql"
	redisstore "iot-platform/internal/store/redis"
)

type Deps struct {
	MySQL  *mysqlstore.Store
	Redis  *redisstore.Store
	AI     *ai.Client
	Logger *slog.Logger
}

type Consumer struct {
	deps  Deps
	cfg   config.MQTTConfig
	c     paho.Client
	queue chan []byte

	processed uint64
	dropped   uint64
	badJSON   uint64
	invalidTK uint64
	aiErr     uint64
	anomalies uint64
}

func NewConsumer(d Deps, cfg config.MQTTConfig) *Consumer {
	return &Consumer{
		deps:  d,
		cfg:   cfg,
		queue: make(chan []byte, cfg.QueueLen),
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	opts := paho.NewClientOptions().
		AddBroker(c.cfg.Broker).
		SetClientID(c.cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true)

	client := paho.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		return tok.Error()
	}
	c.c = client

	// worker pool
	for i := 0; i < c.cfg.Workers; i++ {
		go c.worker(ctx, i)
	}

	// subscribe
	if tok := client.Subscribe(c.cfg.Topic, c.cfg.QoS, c.onMessage); tok.Wait() && tok.Error() != nil {
		return tok.Error()
	}

	// stats ticker (every 5s)
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				p := atomic.SwapUint64(&c.processed, 0)
				d := atomic.SwapUint64(&c.dropped, 0)
				bj := atomic.SwapUint64(&c.badJSON, 0)
				it := atomic.SwapUint64(&c.invalidTK, 0)
				ae := atomic.SwapUint64(&c.aiErr, 0)
				an := atomic.SwapUint64(&c.anomalies, 0)

				c.deps.Logger.Info("stats",
					"processed_5s", p,
					"dropped_5s", d,
					"bad_json_5s", bj,
					"invalid_token_5s", it,
					"ai_err_5s", ae,
					"anomaly_5s", an,
				)
			}
		}
	}()

	<-ctx.Done()
	return nil
}

func (c *Consumer) Close() {
	if c.c != nil && c.c.IsConnected() {
		c.c.Disconnect(200)
	}
}

func (c *Consumer) onMessage(_ paho.Client, msg paho.Message) {
	select {
	case c.queue <- msg.Payload():
	default:
		atomic.AddUint64(&c.dropped, 1)
		// 保留这条WARN方便压测时观察队列是否溢出
		c.deps.Logger.Warn("mqtt queue full, drop")
	}
}

func (c *Consumer) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-c.queue:
			var m domain.DeviceMessage
			if err := json.Unmarshal(b, &m); err != nil {
				atomic.AddUint64(&c.badJSON, 1)
				c.deps.Logger.Warn("bad json", "worker", id, "err", err)
				continue
			}

			// token 校验
			tctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
			dbTok, err := c.deps.MySQL.GetToken(tctx, m.DeviceID)
			cancel()
			if err != nil || dbTok != m.Token {
				atomic.AddUint64(&c.invalidTK, 1)
				c.deps.Logger.Warn("invalid token", "device", m.DeviceID)
				continue
			}

			atomic.AddUint64(&c.processed, 1)

			// 入库
			ictx, cancel := context.WithTimeout(ctx, 1*time.Second)
			_ = c.deps.MySQL.InsertData(ictx, m.DeviceID, m.Temperature, m.Pressure, m.Timestamp)
			cancel()

			// 在线
			rctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
			_ = c.deps.Redis.TouchOnline(rctx, m.DeviceID, time.Now().Unix())
			cancel()

			// AI（失败降级）
			actx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
			anomaly, err := c.deps.AI.Detect(actx, m.Temperature, m.Pressure)
			cancel()
			if err != nil {
				atomic.AddUint64(&c.aiErr, 1)
			}

			if err == nil && anomaly {
				atomic.AddUint64(&c.anomalies, 1)
				c.deps.Logger.Warn("ANOMALY", "device", m.DeviceID, "temp", m.Temperature, "pressure", m.Pressure)
			}
		}
	}
}
