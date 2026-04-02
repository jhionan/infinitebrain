// Package capture implements the note capture domain: creation, retrieval,
// update, soft-delete, and archival of notes owned by an org.
package capture

import (
	"time"

	"github.com/google/uuid"
)

// NoteSource identifies how a note entered the system.
type NoteSource string

// NoteSource values mirror the intake channels supported by the capture pipeline.
const (
	SourceManual   NoteSource = "manual"
	SourceVoice    NoteSource = "voice"
	SourceEmail    NoteSource = "email"
	SourceTelegram NoteSource = "telegram"
	SourceWhatsApp NoteSource = "whatsapp"
	SourceWebhook  NoteSource = "webhook"
)

// NoteStatus tracks where in the AI pipeline a note sits.
type NoteStatus string

// NoteStatus values represent lifecycle stages in the capture-to-classify pipeline.
const (
	StatusInbox      NoteStatus = "inbox"
	StatusClassified NoteStatus = "classified"
	StatusArchived   NoteStatus = "archived"
)

// Visibility controls who can see the note.
// Mirrors the nodes.visibility check constraint.
type Visibility string

// Visibility values mirror the nodes.visibility check constraint in the database.
const (
	VisibilityIndividual   Visibility = "individual"
	VisibilityUnit         Visibility = "unit"
	VisibilityUnitAndAbove Visibility = "unit_and_above"
	VisibilityOrg          Visibility = "org"
	VisibilityPublic       Visibility = "public"
)

// Note is the domain model for a captured note.
// Stored as a row in the nodes table with type = 'note'.
// Source and Status are serialised into the metadata JSONB column.
type Note struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	UserID       uuid.UUID
	Title        string
	Content      string
	Source       NoteSource
	Status       NoteStatus
	PARACategory *string // nil until AI classifies the note
	ProjectID    *uuid.UUID
	Tags         []string
	Visibility   Visibility
	IsPHI        bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ArchivedAt   *time.Time
}

// NoteList is a paginated list of notes.
type NoteList struct {
	Notes    []*Note
	Total    int64
	Page     int
	PageSize int
}

// CreateNoteInput carries the user-supplied fields for note creation.
type CreateNoteInput struct {
	Title      string     // optional
	Content    string     // required
	Source     NoteSource // defaults to SourceManual
	Tags       []string
	Visibility Visibility // defaults to VisibilityIndividual
}

// UpdateNoteInput carries the fields the caller may change.
type UpdateNoteInput struct {
	Title   string
	Content string
	Tags    []string
}
