package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/security"
)

type spyRepo struct {
	recorded []string
}

func (s *spyRepo) RecordHit(_ context.Context, ip, _, _ string) error {
	s.recorded = append(s.recorded, ip)
	return nil
}

func (s *spyRepo) IsBlocked(_ context.Context, _ string) (bool, error) { return false, nil }

func TestHandler_ServeHTTP_Returns404(t *testing.T) {
	h := security.NewHandler(&spyRepo{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestHandler_ServeHTTP_RecordsHit(t *testing.T) {
	spy := &spyRepo{}
	h := security.NewHandler(spy, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if len(spy.recorded) != 1 || spy.recorded[0] != "1.2.3.4" {
		t.Errorf("expected IP recorded, got %v", spy.recorded)
	}
}

func TestNoopRepository_AlwaysReturnsNotBlocked(t *testing.T) {
	var repo security.NoopRepository
	blocked, err := repo.IsBlocked(context.Background(), "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blocked {
		t.Error("noop repo should never return blocked=true")
	}
}
