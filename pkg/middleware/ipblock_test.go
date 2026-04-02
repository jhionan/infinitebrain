package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/pkg/middleware"
)

type stubChecker struct{ blocked bool }

func (s *stubChecker) IsBlocked(_ context.Context, _ string) (bool, error) {
	return s.blocked, nil
}

func TestIPBlocker_AllowsUnblockedIP(t *testing.T) {
	h := middleware.IPBlocker(&stubChecker{blocked: false}, zerolog.Nop())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:0"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestIPBlocker_Blocks403ForBlockedIP(t *testing.T) {
	h := middleware.IPBlocker(&stubChecker{blocked: true}, zerolog.Nop())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:0"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", rr.Code)
	}
}
