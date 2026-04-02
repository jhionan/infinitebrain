package org

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

// Handler handles org HTTP endpoints.
type Handler struct {
	svc    Service
	logger zerolog.Logger
}

// NewHandler creates an org handler.
func NewHandler(svc Service, logger zerolog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// GetOrg handles GET /api/v1/orgs/{slug}
func (h *Handler) GetOrg(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(fmt.Errorf("slug required")))
		return
	}
	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, mapOrgResponse(o))
}

// UpdateOrg handles PUT /api/v1/orgs/{slug}
func (h *Handler) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("no auth claims")))
		return
	}
	slug := r.PathValue("slug")

	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}

	var req struct {
		Name     string      `json:"name"`
		Settings OrgSettings `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(err))
		return
	}

	callerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(err))
		return
	}

	updated, err := h.svc.Update(r.Context(), o.ID, callerID, req.Name, req.Settings)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, mapOrgResponse(updated))
}

// ListMembers handles GET /api/v1/orgs/{slug}/members
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ClaimsFromContext(r.Context()); !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("no auth claims")))
		return
	}
	slug := r.PathValue("slug")
	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	members, err := h.svc.ListMembers(r.Context(), o.ID)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	result := make([]memberResponse, len(members))
	for i, m := range members {
		result[i] = mapMemberResponse(m)
	}
	middleware.JSON(w, http.StatusOK, result)
}

// AddMember handles POST /api/v1/orgs/{slug}/members
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("no auth claims")))
		return
	}
	slug := r.PathValue("slug")
	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(err))
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(fmt.Errorf("invalid user_id")))
		return
	}
	callerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(err))
		return
	}

	if err := h.svc.AddMember(r.Context(), o.ID, userID, req.Role, callerID); err != nil {
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateMemberRole handles PUT /api/v1/orgs/{slug}/members/{userID}
func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("no auth claims")))
		return
	}
	slug := r.PathValue("slug")
	targetIDStr := r.PathValue("userID")

	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(fmt.Errorf("invalid userID")))
		return
	}
	callerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(err))
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(err))
		return
	}

	if err := h.svc.UpdateMemberRole(r.Context(), o.ID, targetID, callerID, req.Role); err != nil {
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember handles DELETE /api/v1/orgs/{slug}/members/{userID}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("no auth claims")))
		return
	}
	slug := r.PathValue("slug")
	targetIDStr := r.PathValue("userID")

	o, err := h.svc.Get(r.Context(), slug)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(fmt.Errorf("invalid userID")))
		return
	}
	callerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(err))
		return
	}

	if err := h.svc.RemoveMember(r.Context(), o.ID, targetID, callerID); err != nil {
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── response shapes ────────────────────────────────────────────────────────────

type orgResponse struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Slug       string      `json:"slug"`
	Plan       string      `json:"plan"`
	MaxMembers *int        `json:"max_members"`
	Settings   OrgSettings `json:"settings"`
	CreatedAt  string      `json:"created_at"`
}

type memberResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	JoinedAt    string `json:"joined_at"`
}

func mapOrgResponse(o *Org) orgResponse {
	return orgResponse{
		ID:         o.ID.String(),
		Name:       o.Name,
		Slug:       o.Slug,
		Plan:       o.Plan,
		MaxMembers: o.MaxMembers,
		Settings:   o.Settings,
		CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func mapMemberResponse(m Member) memberResponse {
	return memberResponse{
		UserID:      m.UserID.String(),
		Email:       m.Email,
		DisplayName: m.DisplayName,
		Role:        m.Role,
		JoinedAt:    m.JoinedAt.Format("2006-01-02T15:04:05Z"),
	}
}
