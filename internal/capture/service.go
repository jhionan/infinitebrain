package capture

import (
	"context"

	"github.com/google/uuid"
)

// NoteService is the business logic contract for the capture domain.
type NoteService interface {
	// Create captures a new note for the authenticated user.
	Create(ctx context.Context, orgID, userID uuid.UUID, in CreateNoteInput) (*Note, error)
	// Get returns a single note. Returns ErrNotFound if the note does not belong to orgID.
	Get(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error)
	// List returns a paginated list of the user's notes.
	List(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error)
	// Inbox returns unclassified notes waiting for AI processing.
	Inbox(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error)
	// Update patches the mutable fields of a note. Only the owner may update.
	Update(ctx context.Context, orgID, callerID, noteID uuid.UUID, in UpdateNoteInput) (*Note, error)
	// Delete soft-deletes a note. Only the owner may delete.
	Delete(ctx context.Context, orgID, callerID, noteID uuid.UUID) error
	// Archive marks a note as archived. Only the owner may archive.
	Archive(ctx context.Context, orgID, callerID, noteID uuid.UUID) (*Note, error)
}
