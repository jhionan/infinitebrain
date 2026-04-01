// pkg/middleware/security_test.go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rian/infinite_brain/pkg/middleware"
)

func TestSecurityHeaders_SetsAllRequiredHeaders(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "no-referrer"},
		{"Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'"},
		{"Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()"},
	}

	h := middleware.SecurityHeaders(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := w.Header().Get(tt.header); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
