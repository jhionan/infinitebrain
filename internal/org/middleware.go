package org

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type orgContextKey struct{}

// OrgResolver extracts the org slug from the request's Host header and injects
// the resolved Org into the request context. Requests with no subdomain (or
// www/api) pass through unchanged — org is then resolved from the JWT claims.
func OrgResolver(repo Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := extractSubdomain(r.Host)
			if slug == "" || slug == "www" || slug == "api" {
				next.ServeHTTP(w, r)
				return
			}

			o, err := repo.FindBySlug(r.Context(), slug)
			if err != nil {
				if errors.Is(err, apperrors.ErrNotFound) {
					http.Error(w, `{"error":{"code":"NOT_FOUND","message":"organization not found"}}`, http.StatusNotFound)
				} else {
					http.Error(w, `{"error":{"code":"INTERNAL","message":"internal server error"}}`, http.StatusInternalServerError)
				}
				return
			}

			ctx := context.WithValue(r.Context(), orgContextKey{}, o)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OrgFromContext returns the Org injected by OrgResolver, or nil.
func OrgFromContext(ctx context.Context) *Org {
	o, _ := ctx.Value(orgContextKey{}).(*Org)
	return o
}

// extractSubdomain returns the leftmost label of host if it has 3+ labels,
// otherwise returns empty string.
// "acme.infinitebrain.io" → "acme"
// "infinitebrain.io" → ""
// "localhost:8080" → ""
func extractSubdomain(host string) string {
	// Strip port if present.
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}
