package redis

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const OnlineZSetKey = "online_devices"

type Store struct {
	rdb *redis.Client
	l   *slog.Logger
}

type Config struct {
	Addr string
	DB   int
}

func New(cfg Config, logger *slog.Logger) (*Store, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		DB:   cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return &Store{rdb: rdb, l: logger}, nil
}

func (s *Store) Close() { _ = s.rdb.Close() }

func (s *Store) TouchOnline(ctx context.Context, deviceID string, ts int64) error {
	return s.rdb.ZAdd(ctx, OnlineZSetKey, &redis.Z{
		Score:  float64(ts),
		Member: deviceID,
	}).Err()
}

// ttlSeconds：例如 10，返回最近 10 秒在线的设备列表
func (s *Store) GetOnline(ctx context.Context, now int64, ttlSeconds int64) ([]string, error) {
	min := now - ttlSeconds
	// 把过旧的删掉
	_ = s.rdb.ZRemRangeByScore(ctx, OnlineZSetKey, "0", strconv.FormatInt(min-1, 10)).Err()

	return s.rdb.ZRangeByScore(ctx, OnlineZSetKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(min, 10),
		Max: strconv.FormatInt(now, 10),
	}).Result()
}
