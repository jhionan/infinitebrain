// Package metrics exposes the Prometheus /metrics endpoint.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns the Prometheus metrics HTTP handler.
// Register custom collectors via prometheus.MustRegister before calling this.
func Handler() http.Handler {
	return promhttp.Handler()
}
