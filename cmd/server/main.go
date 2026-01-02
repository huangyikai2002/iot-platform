package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iot-platform/internal/ai"
	"iot-platform/internal/config"
	"iot-platform/internal/httpapi"
	"iot-platform/internal/mqtt"
	mysqlstore "iot-platform/internal/store/mysql"
	redisstore "iot-platform/internal/store/redis"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 初始化MySQL
	my, err := mysqlstore.New(mysqlstore.Config{
		Host:    cfg.MySQL.Host,
		Port:    cfg.MySQL.Port,
		User:    cfg.MySQL.User,
		Pass:    cfg.MySQL.Pass,
		DB:      cfg.MySQL.DB,
		MaxOpen: cfg.MySQL.MaxOpen,
		MaxIdle: cfg.MySQL.MaxIdle,
	}, logger)
	if err != nil {
		slog.Error("init mysql failed", "err", err)
		os.Exit(1)
	}
	defer my.Close()

	// Redis
	rd, err := redisstore.New(redisstore.Config{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	}, logger)

	if err != nil {
		slog.Error("init redis failed", "err", err)
		os.Exit(1)
	}
	defer rd.Close()

	// AI
	aiCli := ai.New(ai.Config{
		BaseURL:       cfg.AI.BaseURL,
		Timeout:       cfg.AI.Timeout,
		FailThreshold: 5,
		OpenDuration:  20 * time.Second,
	}, logger)

	// HTTP API
	httpSrv := httpapi.New(httpapi.Deps{
		MySQL:  my,
		Redis:  rd,
		Logger: logger,
	}, cfg.HTTP.Addr)

	go func() {
		if err := httpSrv.Start(); err != nil {
			slog.Error("http server error", "err", err)
			stop()
		}
	}()

	// MQTT Consumer
	consumer := mqtt.NewConsumer(mqtt.Deps{
		MySQL:  my,
		Redis:  rd,
		AI:     aiCli,
		Logger: logger,
	}, cfg.MQTT)

	go func() {
		if err := consumer.Run(ctx); err != nil {
			slog.Error("mqtt consumer error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	consumer.Close()

	slog.Info("bye")
}
