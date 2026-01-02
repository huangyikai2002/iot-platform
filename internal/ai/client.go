package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type Config struct {
	BaseURL       string
	Timeout       time.Duration
	FailThreshold int32
	OpenDuration  time.Duration
}

type Client struct {
	baseURL string
	http    *http.Client
	l       *slog.Logger

	failThreshold int32
	openDuration  time.Duration
	failCnt       int32
	openUntilUnix int64
}

func New(cfg Config, logger *slog.Logger) *Client {
	return &Client{
		baseURL:       cfg.BaseURL,
		http:          &http.Client{Timeout: cfg.Timeout},
		l:             logger,
		failThreshold: cfg.FailThreshold,
		openDuration:  cfg.OpenDuration,
	}
}

func (c *Client) isOpen() bool {
	return atomic.LoadInt64(&c.openUntilUnix) > time.Now().Unix()
}

func (c *Client) onOK() {
	atomic.StoreInt32(&c.failCnt, 0)
	atomic.StoreInt64(&c.openUntilUnix, 0)
}

func (c *Client) onFail() {
	n := atomic.AddInt32(&c.failCnt, 1)
	if n >= c.failThreshold {
		atomic.StoreInt64(&c.openUntilUnix, time.Now().Add(c.openDuration).Unix())
	}
}

func (c *Client) Detect(ctx context.Context, temp, pressure float64) (bool, error) {
	if c.isOpen() {
		return false, errors.New("ai circuit open")
	}

	body := map[string]float64{"temperature": temp, "pressure": pressure}
	b, err := json.Marshal(body)
	if err != nil {
		c.onFail()
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/detect", bytes.NewReader(b))
	if err != nil {
		c.onFail()
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		c.onFail()
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		c.onFail()
		return false, errors.New("ai non-2xx")
	}

	var out struct {
		Anomaly bool `json:"anomaly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.onFail()
		return false, err
	}

	c.onOK()
	return out.Anomaly, nil
}
