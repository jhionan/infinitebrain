package capture

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type noteServiceImpl struct {
	repo NoteRepository
}

// NewService returns a NoteService backed by the given repository.
func NewService(repo NoteRepository) NoteService {
	return &noteServiceImpl{repo: repo}
}

func (s *noteServiceImpl) Create(ctx context.Context, orgID, userID uuid.UUID, in CreateNoteInput) (*Note, error) {
	if in.Content == "" {
		return nil, apperrors.ErrValidation.Wrap(errors.New("note content is required"))
	}
	note, err := s.repo.Create(ctx, orgID, userID, in)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	return note, nil
}

func (s *noteServiceImpl) Get(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error) {
	note, err := s.repo.FindByID(ctx, orgID, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note %s: %w", noteID, err)
	}
	return note, nil
}

func (s *noteServiceImpl) List(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error) {
	list, err := s.repo.List(ctx, orgID, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	return list, nil
}

func (s *noteServiceImpl) Inbox(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error) {
	list, err := s.repo.ListInbox(ctx, orgID, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	return list, nil
}

func (s *noteServiceImpl) Update(ctx context.Context, orgID, callerID, noteID uuid.UUID, in UpdateNoteInput) (*Note, error) {
	note, err := s.repo.FindByID(ctx, orgID, noteID)
	if err != nil {
		return nil, fmt.Errorf("find note for update: %w", err)
	}
	if err := s.requireOwner(callerID, note); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, orgID, noteID, in)
	if err != nil {
		return nil, fmt.Errorf("update note %s: %w", noteID, err)
	}
	return updated, nil
}

func (s *noteServiceImpl) Delete(ctx context.Context, orgID, callerID, noteID uuid.UUID) error {
	note, err := s.repo.FindByID(ctx, orgID, noteID)
	if err != nil {
		return fmt.Errorf("find note for delete: %w", err)
	}
	if err := s.requireOwner(callerID, note); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, orgID, noteID); err != nil {
		return fmt.Errorf("delete note %s: %w", noteID, err)
	}
	return nil
}

func (s *noteServiceImpl) Archive(ctx context.Context, orgID, callerID, noteID uuid.UUID) (*Note, error) {
	note, err := s.repo.FindByID(ctx, orgID, noteID)
	if err != nil {
		return nil, fmt.Errorf("find note for archive: %w", err)
	}
	if err := s.requireOwner(callerID, note); err != nil {
		return nil, err
	}
	archived, err := s.repo.Archive(ctx, orgID, noteID)
	if err != nil {
		return nil, fmt.Errorf("archive note %s: %w", noteID, err)
	}
	return archived, nil
}

func (s *noteServiceImpl) requireOwner(callerID uuid.UUID, note *Note) error {
	if note.UserID != callerID {
		return apperrors.ErrForbidden.Wrap(
			errors.New("only the note owner can modify this note"),
		)
	}
	return nil
}
