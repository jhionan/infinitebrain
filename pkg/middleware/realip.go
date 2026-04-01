// Package middleware provides HTTP middleware and response helpers.
package middleware

import (
	"net"
	"net/http"
)

// realIP extracts the client IP from the request.
// It reads RemoteAddr directly — X-Forwarded-For is intentionally NOT trusted
// unless APP_BEHIND_PROXY is configured (future enhancement), to prevent IP spoofing.
func realIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
