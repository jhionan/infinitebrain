package org

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

// InviteHandler handles HTTP requests for the org invite flow.
type InviteHandler struct {
	svc    InviteService
	logger zerolog.Logger
}

// NewInviteHandler creates an InviteHandler.
func NewInviteHandler(svc InviteService, logger zerolog.Logger) *InviteHandler {
	return &InviteHandler{svc: svc, logger: logger}
}

type createInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// CreateInvite handles POST /api/v1/orgs/{slug}/invites.
func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("authentication required")))
		return
	}
	o := OrgFromContext(r.Context())
	if o == nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("org context missing")))
		return
	}
	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}
	if req.Email == "" || req.Role == "" {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(fmt.Errorf("email and role are required")))
		return
	}
	invite, err := h.svc.CreateInvite(r.Context(), o.ID, req.Email, req.Role, claims.UserID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("create invite failed")
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusCreated, invite)
}

// AcceptInvite handles POST /api/v1/invites/{token}/accept.
func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("authentication required")))
		return
	}
	token := r.PathValue("token")
	if token == "" {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("token is required")))
		return
	}
	if err := h.svc.AcceptInvite(r.Context(), token, claims.UserID); err != nil {
		h.logger.Warn().Err(err).Msg("accept invite failed")
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
