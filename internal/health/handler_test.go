package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rian/infinite_brain/internal/health"
)

func TestLiveHandler_Returns200(t *testing.T) {
	checker := health.NewChecker()
	h := health.NewHandler(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()
	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

func TestReadyHandler_AllHealthy_Returns200(t *testing.T) {
	checker := health.NewChecker(
		health.WithProbe("db", &fakeProbe{}),
	)
	h := health.NewHandler(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReadyHandler_UnhealthyProbe_Returns503(t *testing.T) {
	checker := health.NewChecker(
		health.WithProbe("db", &fakeProbe{err: errFake}),
	)
	h := health.NewHandler(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()
	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
