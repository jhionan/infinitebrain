package capture_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/capture"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// ── mock repository ────────────────────────────────────────────────────────────

type mockNoteRepo struct {
	notes map[uuid.UUID]*capture.Note
}

func newMockRepo() *mockNoteRepo {
	return &mockNoteRepo{notes: make(map[uuid.UUID]*capture.Note)}
}

func (m *mockNoteRepo) Create(_ context.Context, orgID, userID uuid.UUID, in capture.CreateNoteInput) (*capture.Note, error) {
	if in.Content == "" {
		return nil, apperrors.ErrValidation.Wrap(errors.New("content required"))
	}
	n := &capture.Note{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		Content:   in.Content,
		Title:     in.Title,
		Source:    in.Source,
		Status:    capture.StatusInbox,
		Tags:      in.Tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.notes[n.ID] = n
	return n, nil
}

func (m *mockNoteRepo) FindByID(_ context.Context, orgID, noteID uuid.UUID) (*capture.Note, error) {
	n, ok := m.notes[noteID]
	if !ok || n.OrgID != orgID {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("note not found"))
	}
	return n, nil
}

func (m *mockNoteRepo) List(_ context.Context, orgID, userID uuid.UUID, page, pageSize int) (*capture.NoteList, error) {
	var notes []*capture.Note
	for _, n := range m.notes {
		if n.OrgID == orgID && n.UserID == userID {
			notes = append(notes, n)
		}
	}
	return &capture.NoteList{Notes: notes, Total: int64(len(notes)), Page: page, PageSize: pageSize}, nil
}

func (m *mockNoteRepo) ListInbox(_ context.Context, orgID, userID uuid.UUID, page, pageSize int) (*capture.NoteList, error) {
	var notes []*capture.Note
	for _, n := range m.notes {
		if n.OrgID == orgID && n.UserID == userID && n.Status == capture.StatusInbox {
			notes = append(notes, n)
		}
	}
	return &capture.NoteList{Notes: notes, Total: int64(len(notes)), Page: page, PageSize: pageSize}, nil
}

func (m *mockNoteRepo) Update(_ context.Context, orgID, noteID uuid.UUID, in capture.UpdateNoteInput) (*capture.Note, error) {
	n, ok := m.notes[noteID]
	if !ok || n.OrgID != orgID {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("note not found"))
	}
	n.Title = in.Title
	n.Content = in.Content
	n.Tags = in.Tags
	return n, nil
}

func (m *mockNoteRepo) Delete(_ context.Context, orgID, noteID uuid.UUID) error {
	n, ok := m.notes[noteID]
	if !ok || n.OrgID != orgID {
		return apperrors.ErrNotFound.Wrap(errors.New("note not found"))
	}
	delete(m.notes, noteID)
	return nil
}

func (m *mockNoteRepo) Archive(_ context.Context, orgID, noteID uuid.UUID) (*capture.Note, error) {
	n, ok := m.notes[noteID]
	if !ok || n.OrgID != orgID {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("note not found"))
	}
	now := time.Now()
	n.ArchivedAt = &now
	n.Status = capture.StatusArchived
	return n, nil
}

// ── tests ──────────────────────────────────────────────────────────────────────

func newTestSvc() (capture.NoteService, *mockNoteRepo) {
	repo := newMockRepo()
	return capture.NewService(repo), repo
}

func TestNoteService_Create_ValidInput_Succeeds(t *testing.T) {
	svc, _ := newTestSvc()
	orgID, userID := uuid.New(), uuid.New()

	note, err := svc.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Test note.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if note.Status != capture.StatusInbox {
		t.Errorf("Status = %q, want inbox", note.Status)
	}
}

func TestNoteService_Create_EmptyContent_ReturnsValidation(t *testing.T) {
	svc, _ := newTestSvc()
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), capture.CreateNoteInput{Content: ""})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestNoteService_Update_WrongOwner_ReturnsForbidden(t *testing.T) {
	svc, repo := newTestSvc()
	orgID, ownerID := uuid.New(), uuid.New()
	note, _ := repo.Create(context.Background(), orgID, ownerID, capture.CreateNoteInput{Content: "owned"})

	callerID := uuid.New() // different user
	_, err := svc.Update(context.Background(), orgID, callerID, note.ID, capture.UpdateNoteInput{Content: "hacked"})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestNoteService_Delete_WrongOwner_ReturnsForbidden(t *testing.T) {
	svc, repo := newTestSvc()
	orgID, ownerID := uuid.New(), uuid.New()
	note, _ := repo.Create(context.Background(), orgID, ownerID, capture.CreateNoteInput{Content: "owned"})

	callerID := uuid.New()
	err := svc.Delete(context.Background(), orgID, callerID, note.ID)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestNoteService_Archive_WrongOwner_ReturnsForbidden(t *testing.T) {
	svc, repo := newTestSvc()
	orgID, ownerID := uuid.New(), uuid.New()
	note, _ := repo.Create(context.Background(), orgID, ownerID, capture.CreateNoteInput{Content: "owned"})

	callerID := uuid.New()
	_, err := svc.Archive(context.Background(), orgID, callerID, note.ID)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestNoteService_Get_NotFound_ReturnsError(t *testing.T) {
	svc, _ := newTestSvc()
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
