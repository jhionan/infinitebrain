// pkg/middleware/cors_test.go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rian/infinite_brain/pkg/middleware"
)

func TestCORS_AllowsPermittedOrigin(t *testing.T) {
	h := middleware.CORS([]string{"https://app.example.com"})(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("ACAO = %q, want origin", got)
	}
}

func TestCORS_BlocksUnknownOrigin(t *testing.T) {
	h := middleware.CORS([]string{"https://app.example.com"})(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty for blocked origin", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	h := middleware.CORS([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w.Code)
	}
}
