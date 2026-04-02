package middleware

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

// IPChecker is satisfied by security.HoneypotRepository (and test mocks).
type IPChecker interface {
	IsBlocked(ctx context.Context, ip string) (bool, error)
}

// IPBlocker returns middleware that rejects requests from blocked IPs with 403.
func IPBlocker(checker IPChecker, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			blocked, err := checker.IsBlocked(r.Context(), ip)
			if err != nil {
				logger.Error().Err(err).Str("ip", ip).Msg("ip block check failed")
			}
			if blocked {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
