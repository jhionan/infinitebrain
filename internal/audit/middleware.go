package audit

import (
	"net/http"
	"strings"
)

// Middleware returns HTTP middleware that auto-logs all state-changing requests
// (POST, PUT, PATCH, DELETE) that return a 2xx status, using claims from context.
// Non-mutating requests (GET, HEAD, OPTIONS) are skipped.
func Middleware(recorder Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutating(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			rr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rr, r)
			if rr.status >= 200 && rr.status < 300 {
				recorder.Record(r.Context(), r.Method+" "+r.URL.Path, "", nil, nil, nil)
			}
		})
	}
}

func isMutating(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
