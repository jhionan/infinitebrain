package capture

import (
	"net/http"

	"github.com/rian/infinite_brain/internal/auth"
)

// RegisterRoutes wires the capture domain routes into mux.
// authed wraps a handler with JWT authentication middleware.
func RegisterRoutes(mux *http.ServeMux, h *Handler, authed func(http.Handler) http.Handler) {
	requireCreate := auth.Require(auth.PermCreateNode)
	requireRead := auth.Require(auth.PermReadNode)
	requireEdit := auth.Require(auth.PermEditOwnNode)
	requireDelete := auth.Require(auth.PermDeleteOwnNode)

	mux.HandleFunc("POST /api/v1/notes",
		authed(requireCreate(http.HandlerFunc(h.Create))).ServeHTTP)
	mux.HandleFunc("GET /api/v1/notes",
		authed(requireRead(http.HandlerFunc(h.List))).ServeHTTP)
	mux.HandleFunc("GET /api/v1/notes/{id}",
		authed(requireRead(http.HandlerFunc(h.Get))).ServeHTTP)
	mux.HandleFunc("PATCH /api/v1/notes/{id}",
		authed(requireEdit(http.HandlerFunc(h.Update))).ServeHTTP)
	mux.HandleFunc("DELETE /api/v1/notes/{id}",
		authed(requireDelete(http.HandlerFunc(h.Delete))).ServeHTTP)
	mux.HandleFunc("GET /api/v1/inbox",
		authed(requireRead(http.HandlerFunc(h.Inbox))).ServeHTTP)
	mux.HandleFunc("POST /api/v1/notes/{id}/archive",
		authed(requireEdit(http.HandlerFunc(h.Archive))).ServeHTTP)
}
