// pkg/metrics/metrics_test.go
package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rian/infinite_brain/pkg/metrics"
)

func TestHandler_Returns200WithTextPlain(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metrics.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain prefix", ct)
	}
	if !strings.Contains(w.Body.String(), "go_goroutines") {
		t.Error("expected go_goroutines metric in response")
	}
}
