package capture

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/middleware"
)

// Handler handles HTTP requests for the capture domain.
type Handler struct {
	svc    NoteService
	logger zerolog.Logger
}

// NewHandler creates a capture Handler.
func NewHandler(svc NoteService, logger zerolog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// createNoteBody is the JSON payload for note creation.
// Identical field set to CreateNoteInput but with JSON struct tags.
type createNoteBody struct {
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Source     NoteSource `json:"source"`
	Tags       []string   `json:"tags"`
	Visibility Visibility `json:"visibility"`
}

// updateNoteBody is the JSON payload for note updates.
// Identical field set to UpdateNoteInput but with JSON struct tags.
type updateNoteBody struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// mustClaims extracts JWT claims from the request context.
// Writes a 401 response and returns false if claims are absent.
func (h *Handler) mustClaims(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	c, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, apperrors.ErrUnauthorized)
		return nil, false
	}
	return c, true
}

// Create handles POST /api/v1/notes.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	var body createNoteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}
	note, err := h.svc.Create(r.Context(), claims.OrgID, claims.UserID, CreateNoteInput(body))
	if err != nil {
		h.logger.Warn().Err(err).Msg("create note failed")
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusCreated, note)
}

// Get handles GET /api/v1/notes/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("invalid note id")))
		return
	}
	note, err := h.svc.Get(r.Context(), claims.OrgID, noteID)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, note)
}

// List handles GET /api/v1/notes.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	page, pageSize := parsePagination(r)
	list, err := h.svc.List(r.Context(), claims.OrgID, claims.UserID, page, pageSize)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, list)
}

// Inbox handles GET /api/v1/inbox.
func (h *Handler) Inbox(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	page, pageSize := parsePagination(r)
	list, err := h.svc.Inbox(r.Context(), claims.OrgID, claims.UserID, page, pageSize)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, list)
}

// Update handles PATCH /api/v1/notes/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("invalid note id")))
		return
	}
	var body updateNoteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(err))
		return
	}
	note, err := h.svc.Update(r.Context(), claims.OrgID, claims.UserID, noteID, UpdateNoteInput(body))
	if err != nil {
		h.logger.Warn().Err(err).Msg("update note failed")
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, note)
}

// Delete handles DELETE /api/v1/notes/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("invalid note id")))
		return
	}
	if err := h.svc.Delete(r.Context(), claims.OrgID, claims.UserID, noteID); err != nil {
		middleware.JSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Archive handles POST /api/v1/notes/{id}/archive.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.mustClaims(w, r)
	if !ok {
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		middleware.JSONError(w, apperrors.ErrBadRequest.Wrap(fmt.Errorf("invalid note id")))
		return
	}
	note, err := h.svc.Archive(r.Context(), claims.OrgID, claims.UserID, noteID)
	if err != nil {
		middleware.JSONError(w, err)
		return
	}
	middleware.JSON(w, http.StatusOK, note)
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
