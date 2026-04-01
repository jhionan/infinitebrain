package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"

	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

// Handler handles HTTP requests for the auth domain.
type Handler struct {
	svc    Service
	logger zerolog.Logger
}

// NewHandler creates an auth Handler.
func NewHandler(svc Service, logger zerolog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}
	if req.Email == "" || req.DisplayName == "" || req.Password == "" {
		middleware.JSONError(w, apperrors.ErrValidation.Wrap(
			fmt.Errorf("email, display_name, and password are required"),
		))
		return
	}

	pair, err := h.svc.Register(r.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		h.logger.Warn().Err(err).Str("email", req.Email).Msg("register failed")
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusCreated, pair)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, pair)
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, pair)
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/v1/auth/me — requires Auth middleware to have set claims.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("authentication required")))
		return
	}

	profile, err := h.svc.Me(r.Context(), claims.UserID.String())
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, profile)
}
