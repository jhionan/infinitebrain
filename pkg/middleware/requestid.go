// Package middleware provides HTTP middleware and response helpers.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey int

const requestIDKey contextKey = iota

// RequestID injects a unique request ID into the context and response header.
// Propagates an existing X-Request-ID header if present.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext retrieves the request ID set by the RequestID middleware.
// Returns empty string if not present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
