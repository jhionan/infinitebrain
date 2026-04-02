// Package security implements honeypot detection and IP blocking.
package security

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

// HoneypotRepository persists honeypot hits and blocked IPs.
// The noop implementation is used until pgxpool is wired in Task 10.
type HoneypotRepository interface {
	RecordHit(ctx context.Context, ip, path, userAgent string) error
	IsBlocked(ctx context.Context, ip string) (bool, error)
}

// NoopRepository satisfies HoneypotRepository without a database.
type NoopRepository struct{}

func (NoopRepository) RecordHit(_ context.Context, _, _, _ string) error  { return nil }
func (NoopRepository) IsBlocked(_ context.Context, _ string) (bool, error) { return false, nil }

// Handler is the honeypot HTTP handler.
type Handler struct {
	repo   HoneypotRepository
	logger zerolog.Logger
}

// NewHandler returns a Handler backed by repo.
func NewHandler(repo HoneypotRepository, logger zerolog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// ServeHTTP records the hit and returns 404 so scanners learn nothing.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	h.logger.Warn().
		Str("ip", ip).
		Str("path", r.URL.Path).
		Str("user_agent", r.UserAgent()).
		Msg("honeypot triggered")
	if err := h.repo.RecordHit(r.Context(), ip, r.URL.Path, r.UserAgent()); err != nil {
		h.logger.Error().Err(err).Msg("failed to record honeypot hit")
	}
	http.NotFound(w, r)
}
