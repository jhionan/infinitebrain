package capture

import (
	"context"

	"github.com/google/uuid"
)

// NoteRepository is the data access contract for the capture domain.
// All implementations must scope every query to orgID to enforce tenant isolation.
type NoteRepository interface {
	// Create inserts a new note and returns the persisted record.
	Create(ctx context.Context, orgID, userID uuid.UUID, in CreateNoteInput) (*Note, error)
	// FindByID returns a note visible to the given org.
	FindByID(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error)
	// List returns a page of notes for the user, newest first.
	List(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error)
	// ListInbox returns a page of unclassified (status=inbox) notes.
	ListInbox(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error)
	// Update patches the mutable fields of a note.
	Update(ctx context.Context, orgID, noteID uuid.UUID, in UpdateNoteInput) (*Note, error)
	// Delete soft-deletes a note (sets deleted_at).
	Delete(ctx context.Context, orgID, noteID uuid.UUID) error
	// Archive marks a note as archived and sets status=archived in metadata.
	Archive(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error)
}
