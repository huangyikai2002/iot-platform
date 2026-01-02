package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	MySQL MySQLConfig
	Redis RedisConfig
	MQTT  MQTTConfig
	HTTP  HTTPConfig
	AI    AIConfig
}

type MySQLConfig struct {
	Host, Port, User, Pass, DB string
	MaxOpen, MaxIdle           int
}

type RedisConfig struct {
	Addr string
	DB   int
}

type MQTTConfig struct {
	Broker   string
	ClientID string
	Topic    string
	QoS      byte
	Workers  int
	QueueLen int
}

type HTTPConfig struct {
	Addr string
}

type AIConfig struct {
	BaseURL string
	Timeout time.Duration
}

func Load() Config {
	return Config{
		MySQL: MySQLConfig{
			Host:    getenv("MYSQL_HOST", "127.0.0.1"),
			Port:    getenv("MYSQL_PORT", "3306"),
			User:    getenv("MYSQL_USER", "root"),
			Pass:    getenv("MYSQL_PASS", "root"),
			DB:      getenv("MYSQL_DB", "iot_platform"),
			MaxOpen: getenvInt("MYSQL_MAX_OPEN", 50),
			MaxIdle: getenvInt("MYSQL_MAX_IDLE", 10),
		},
		Redis: RedisConfig{
			Addr: getenv("REDIS_ADDR", "127.0.0.1:6379"),
			DB:   getenvInt("REDIS_DB", 0),
		},
		MQTT: MQTTConfig{
			Broker:   getenv("MQTT_BROKER", "tcp://127.0.0.1:1883"),
			ClientID: getenv("MQTT_CLIENT_ID", "go_server"),
			Topic:    getenv("MQTT_TOPIC", "devices/+/data"),
			QoS:      byte(getenvInt("MQTT_QOS", 1)),
			Workers:  getenvInt("WORKERS", 8),
			QueueLen: getenvInt("QUEUE_LEN", 4096),
		},
		HTTP: HTTPConfig{
			Addr: getenv("HTTP_ADDR", ":8080"),
		},
		AI: AIConfig{
			BaseURL: getenv("AI_BASE_URL", "http://127.0.0.1:5001"),
			Timeout: time.Duration(getenvInt("AI_TIMEOUT_MS", 900)) * time.Millisecond,
		},
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
