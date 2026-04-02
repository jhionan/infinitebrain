package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

type contextKey int

const claimsKey contextKey = iota

// Auth is HTTP middleware that extracts and validates the Bearer JWT,
// injecting Claims into the request context.
// Returns 401 if the token is missing or invalid.
func Auth(signer *Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(errors.New("missing authorization header")))
				return
			}
			claims, err := signer.Verify(token)
			if err != nil {
				middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(err))
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves JWT claims injected by the Auth middleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// ContextWithClaims returns a copy of ctx with claims injected.
// Used in tests and packages (like audit) that need a claims-bearing context
// without going through the Auth middleware's HTTP layer.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// extractBearer returns the token from a "Bearer <token>" header, or empty string.
func extractBearer(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
