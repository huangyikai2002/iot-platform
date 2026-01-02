package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	mysqlstore "iot-platform/internal/store/mysql"
	redisstore "iot-platform/internal/store/redis"
)

type Deps struct {
	MySQL  *mysqlstore.Store
	Redis  *redisstore.Store
	Logger *slog.Logger
}

type Server struct {
	deps Deps
	srv  *http.Server
}

func New(d Deps, addr string) *Server {
	mux := http.NewServeMux()

	h := &handlers{deps: d}
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/devices/register", h.register)
	mux.HandleFunc("/devices/online", h.online)
	mux.HandleFunc("/devices/", h.latest) // /devices/{id}/latest

	return &Server{
		deps: d,
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	err := s.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
