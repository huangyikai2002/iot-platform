package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
	l  *slog.Logger

	stmtInsertData  *sql.Stmt
	stmtSelectToken *sql.Stmt
	stmtInsertDev   *sql.Stmt
	stmtGetLatest   *sql.Stmt
	stmtGetDevToken *sql.Stmt
}

type Config struct {
	Host, Port, User, Pass, DB string
	MaxOpen, MaxIdle           int
}

func New(cfg Config, logger *slog.Logger) (*Store, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.DB)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db, l: logger}
	if err := s.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.prepare(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s.stmtInsertData != nil {
		_ = s.stmtInsertData.Close()
	}
	if s.stmtSelectToken != nil {
		_ = s.stmtSelectToken.Close()
	}
	if s.stmtInsertDev != nil {
		_ = s.stmtInsertDev.Close()
	}
	if s.stmtGetLatest != nil {
		_ = s.stmtGetLatest.Close()
	}
	if s.stmtGetDevToken != nil {
		_ = s.stmtGetDevToken.Close()
	}
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *Store) ensureSchema() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS devices (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		device_id VARCHAR(64) NOT NULL UNIQUE,
		token VARCHAR(128) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS device_data (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		device_id VARCHAR(64) NOT NULL,
		temperature DOUBLE NOT NULL,
		pressure DOUBLE NOT NULL,
		timestamp BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_device_time (device_id, id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`)
	return err
}

func (s *Store) prepare() error {
	var err error
	s.stmtInsertData, err = s.db.Prepare(`INSERT INTO device_data (device_id, temperature, pressure, timestamp) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	s.stmtSelectToken, err = s.db.Prepare(`SELECT token FROM devices WHERE device_id=?`)
	if err != nil {
		return err
	}
	s.stmtInsertDev, err = s.db.Prepare(`INSERT INTO devices (device_id, token) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	s.stmtGetLatest, err = s.db.Prepare(`
		SELECT temperature, pressure, timestamp
		FROM device_data
		WHERE device_id=?
		ORDER BY id DESC
		LIMIT 1`)
	if err != nil {
		return err
	}
	s.stmtGetDevToken, err = s.db.Prepare(`SELECT token FROM devices WHERE device_id=?`)
	return err
}

func (s *Store) InsertData(ctx context.Context, deviceID string, t, p float64, ts int64) error {
	_, err := s.stmtInsertData.ExecContext(ctx, deviceID, t, p, ts)
	return err
}

func (s *Store) GetToken(ctx context.Context, deviceID string) (string, error) {
	var tok string
	err := s.stmtSelectToken.QueryRowContext(ctx, deviceID).Scan(&tok)
	return tok, err
}

func (s *Store) GetLatest(ctx context.Context, deviceID string) (temp, pressure float64, ts int64, err error) {
	err = s.stmtGetLatest.QueryRowContext(ctx, deviceID).Scan(&temp, &pressure, &ts)
	return
}

func (s *Store) RegisterDevice(ctx context.Context) (deviceID, token string, err error) {
	deviceID, err = genID()
	if err != nil {
		return "", "", err
	}
	token, err = genToken(16)
	if err != nil {
		return "", "", err
	}
	_, err = s.stmtInsertDev.ExecContext(ctx, deviceID, token)
	return
}

func genToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func genID() (string, error) {
	t, err := genToken(6)
	if err != nil {
		return "", err
	}
	return "device_" + t, nil
}
