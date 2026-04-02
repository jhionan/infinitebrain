package health

import (
	"encoding/json"
	"net/http"
)

// HTTPHandler serves the /health/live and /health/ready endpoints.
type HTTPHandler struct {
	checker *Checker
}

// NewHandler creates an HTTPHandler that delegates to the provided Checker.
func NewHandler(c *Checker) *HTTPHandler {
	return &HTTPHandler{checker: c}
}

// Live handles GET /health/live.
// Returns 200 if the process is running.
func (h *HTTPHandler) Live(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Ready handles GET /health/ready.
// Returns 200 if all probes pass, 503 otherwise.
func (h *HTTPHandler) Ready(w http.ResponseWriter, r *http.Request) {
	result := h.checker.Ready(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if result.OK {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	status := "ok"
	if !result.OK {
		status = "degraded"
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": result.Checks,
	})
}
