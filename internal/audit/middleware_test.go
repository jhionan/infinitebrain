package audit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/audit"
)

type stubRecorder struct {
	recorded []string
}

func (s *stubRecorder) Record(_ context.Context, action, _ string, _ *uuid.UUID, _, _ any) {
	s.recorded = append(s.recorded, action)
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func createdHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func badRequestHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func TestAuditMiddleware_RecordsMutatingRequestsOn2xx(t *testing.T) {
	rec := &stubRecorder{}
	mw := audit.Middleware(rec)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec.recorded = nil
		req := httptest.NewRequest(method, "/api/v1/something", nil)
		rr := httptest.NewRecorder()
		mw(http.HandlerFunc(createdHandler)).ServeHTTP(rr, req)
		if len(rec.recorded) != 1 {
			t.Errorf("method %s: expected 1 record call, got %d", method, len(rec.recorded))
		}
	}
}

func TestAuditMiddleware_SkipsNonMutatingRequests(t *testing.T) {
	rec := &stubRecorder{}
	mw := audit.Middleware(rec)

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		rec.recorded = nil
		req := httptest.NewRequest(method, "/api/v1/something", nil)
		rr := httptest.NewRecorder()
		mw(http.HandlerFunc(okHandler)).ServeHTTP(rr, req)
		if len(rec.recorded) != 0 {
			t.Errorf("method %s: expected 0 record calls, got %d", method, len(rec.recorded))
		}
	}
}

func TestAuditMiddleware_SkipsNon2xxResponses(t *testing.T) {
	rec := &stubRecorder{}
	mw := audit.Middleware(rec)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/something", nil)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(badRequestHandler)).ServeHTTP(rr, req)

	if len(rec.recorded) != 0 {
		t.Errorf("expected 0 record calls for 4xx, got %d", len(rec.recorded))
	}
}

func TestAuditMiddleware_PassesThroughToNextHandler(t *testing.T) {
	rec := &stubRecorder{}
	mw := audit.Middleware(rec)
	called := false

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("expected next handler to be called")
	}
}
