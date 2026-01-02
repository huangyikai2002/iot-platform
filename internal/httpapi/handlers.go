package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type handlers struct {
	deps Deps
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (h *handlers) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"error": "method not allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	id, tok, err := h.deps.MySQL.RegisterDevice(ctx)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "register failed"})
		return
	}
	writeJSON(w, 200, map[string]any{"device_id": id, "token": tok})
}

func (h *handlers) online(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
	defer cancel()

	// ttl 默认 10 秒
	devs, err := h.deps.Redis.GetOnline(ctx, now, 10)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "redis error"})
		return
	}
	writeJSON(w, 200, map[string]any{"online_devices": devs})
}

func (h *handlers) latest(w http.ResponseWriter, r *http.Request) {
	// /devices/{id}/latest
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "devices" || parts[2] != "latest" {
		writeJSON(w, 400, map[string]any{"error": "invalid path"})
		return
	}
	deviceID := parts[1]

	ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
	defer cancel()
	temp, pressure, ts, err := h.deps.MySQL.GetLatest(ctx, deviceID)
	if err != nil {
		writeJSON(w, 404, map[string]any{"error": "no data"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"device_id":   deviceID,
		"temperature": temp,
		"pressure":    pressure,
		"timestamp":   ts,
	})
}
