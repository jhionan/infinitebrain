package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// RateLimiter is the interface satisfied by cache.Store (and test mocks).
type RateLimiter interface {
	IncrWithExpire(key string, ttl time.Duration) int
}

// RateLimit returns a middleware that limits each client IP to limit requests
// per window using a sliding-window counter. Requests over the limit receive
// 429 Too Many Requests.
func RateLimit(store RateLimiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("rl:%s", ip)
			if store.IncrWithExpire(key, window) > limit {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
