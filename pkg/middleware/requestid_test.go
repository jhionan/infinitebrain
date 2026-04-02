// Package middleware_test contains tests for the middleware package.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rian/infinite_brain/pkg/middleware"
)

func TestRequestID_GeneratesIDWhenMissing(t *testing.T) {
	var capturedID string
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if capturedID == "" {
		t.Fatal("expected request ID in context, got empty string")
	}
	if w.Header().Get("X-Request-ID") != capturedID {
		t.Errorf("response X-Request-ID = %q, want %q", w.Header().Get("X-Request-ID"), capturedID)
	}
}

func TestRequestID_PropagatesExistingID(t *testing.T) {
	const want = "trace-abc-123"
	var got string
	h := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", want)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got != want {
		t.Errorf("context ID = %q, want %q", got, want)
	}
	if w.Header().Get("X-Request-ID") != want {
		t.Errorf("response X-Request-ID = %q, want %q", w.Header().Get("X-Request-ID"), want)
	}
}
