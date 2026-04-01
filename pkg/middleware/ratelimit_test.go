package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rian/infinite_brain/pkg/middleware"
)

// stubLimiter is a minimal in-memory stub that satisfies middleware.RateLimiter.
type stubLimiter struct {
	counts map[string]int
}

func newStubLimiter() *stubLimiter {
	return &stubLimiter{counts: make(map[string]int)}
}

func (s *stubLimiter) IncrWithExpire(key string, _ time.Duration) int {
	s.counts[key]++
	return s.counts[key]
}

func TestRateLimit_AllowsRequestsUnderLimit(t *testing.T) {
	stub := newStubLimiter()
	h := middleware.RateLimit(stub, 3, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5000"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i+1, rr.Code)
		}
	}
}

func TestRateLimit_Blocks429WhenLimitExceeded(t *testing.T) {
	stub := newStubLimiter()
	h := middleware.RateLimit(stub, 2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5000"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("warm-up: got %d, want 200", rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5000"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("got %d, want 429", rr.Code)
	}
}

func TestRateLimit_TracksIPsSeparately(t *testing.T) {
	stub := newStubLimiter()
	h := middleware.RateLimit(stub, 1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, ip := range []string{"1.1.1.1:0", "2.2.2.2:0"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("IP %s: got %d, want 200", ip, rr.Code)
		}
	}
}
