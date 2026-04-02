package auth

import (
	"fmt"
	"net/http"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

// Require returns HTTP middleware that checks the authenticated caller's role
// has the given permission. Must be placed after the Auth middleware in the chain.
// Returns 401 if no claims are present; 403 if the role lacks the permission.
func Require(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(
					fmt.Errorf("authentication required"),
				))
				return
			}
			if !Can(claims.Role, perm) {
				middleware.JSONError(w, apperrors.ErrForbidden.WithMessage(
					fmt.Sprintf("role %q cannot perform %q", claims.Role, perm),
				))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
