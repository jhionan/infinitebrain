package capture_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/capture"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// mockNoteService stubs NoteService for handler tests.
type mockNoteService struct {
	note *capture.Note
	list *capture.NoteList
	err  error
}

func (m *mockNoteService) Create(_ context.Context, _, _ uuid.UUID, _ capture.CreateNoteInput) (*capture.Note, error) {
	return m.note, m.err
}

func (m *mockNoteService) Get(_ context.Context, _, _ uuid.UUID) (*capture.Note, error) {
	return m.note, m.err
}

func (m *mockNoteService) List(_ context.Context, _, _ uuid.UUID, _, _ int) (*capture.NoteList, error) {
	return m.list, m.err
}

func (m *mockNoteService) Inbox(_ context.Context, _, _ uuid.UUID, _, _ int) (*capture.NoteList, error) {
	return m.list, m.err
}

func (m *mockNoteService) Update(_ context.Context, _, _, _ uuid.UUID, _ capture.UpdateNoteInput) (*capture.Note, error) {
	return m.note, m.err
}

func (m *mockNoteService) Delete(_ context.Context, _, _, _ uuid.UUID) error {
	return m.err
}

func (m *mockNoteService) Archive(_ context.Context, _, _, _ uuid.UUID) (*capture.Note, error) {
	return m.note, m.err
}

// withClaims injects JWT claims into the request context (mirrors auth middleware).
func withClaims(r *http.Request, orgID, userID uuid.UUID) *http.Request {
	claims := &auth.Claims{OrgID: orgID, UserID: userID, Role: "editor"}
	return r.WithContext(auth.ContextWithClaims(r.Context(), claims))
}

func newTestNoteHandler(svc capture.NoteService) *capture.Handler {
	return capture.NewHandler(svc, zerolog.Nop())
}

func TestNoteHandler_Create_ValidBody_Returns201(t *testing.T) {
	noteID := uuid.New()
	svc := &mockNoteService{note: &capture.Note{
		ID:      noteID,
		Content: "hello",
		Status:  capture.StatusInbox,
		Tags:    []string{},
	}}
	h := newTestNoteHandler(svc)

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(body))
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestNoteHandler_Create_MissingContent_Returns422(t *testing.T) {
	svc := &mockNoteService{err: apperrors.ErrValidation.Wrap(errors.New("content required"))}
	h := newTestNoteHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{}`))
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestNoteHandler_Create_NoClaims_Returns401(t *testing.T) {
	h := newTestNoteHandler(&mockNoteService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"x"}`))
	// No claims injected.
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestNoteHandler_Get_ValidID_Returns200(t *testing.T) {
	noteID := uuid.New()
	svc := &mockNoteService{note: &capture.Note{ID: noteID, Content: "found", Tags: []string{}}}
	h := newTestNoteHandler(svc)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/notes/%s", noteID), nil)
	req.SetPathValue("id", noteID.String())
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestNoteHandler_Get_NotFound_Returns404(t *testing.T) {
	svc := &mockNoteService{err: apperrors.ErrNotFound.Wrap(errors.New("note not found"))}
	h := newTestNoteHandler(svc)

	noteID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/notes/%s", noteID), nil)
	req.SetPathValue("id", noteID.String())
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestNoteHandler_List_Returns200(t *testing.T) {
	svc := &mockNoteService{list: &capture.NoteList{Notes: []*capture.Note{}, Total: 0, Page: 1, PageSize: 20}}
	h := newTestNoteHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestNoteHandler_Delete_ValidID_Returns204(t *testing.T) {
	svc := &mockNoteService{}
	h := newTestNoteHandler(svc)

	noteID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/notes/%s", noteID), nil)
	req.SetPathValue("id", noteID.String())
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestNoteHandler_Archive_ValidID_Returns200(t *testing.T) {
	noteID := uuid.New()
	svc := &mockNoteService{note: &capture.Note{ID: noteID, Status: capture.StatusArchived, Tags: []string{}}}
	h := newTestNoteHandler(svc)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notes/%s/archive", noteID), nil)
	req.SetPathValue("id", noteID.String())
	req = withClaims(req, uuid.New(), uuid.New())
	w := httptest.NewRecorder()

	h.Archive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
