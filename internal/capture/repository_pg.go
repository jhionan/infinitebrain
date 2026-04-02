package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlcdb "github.com/rian/infinite_brain/db/sqlc"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type pgNoteRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcdb.Queries
}

// NewRepository returns a PostgreSQL-backed NoteRepository.
func NewRepository(pool *pgxpool.Pool) NoteRepository {
	return &pgNoteRepository{pool: pool, queries: sqlcdb.New(pool)}
}

// noteMetadata is the shape stored in the nodes.metadata JSONB column for notes.
type noteMetadata struct {
	Source NoteSource `json:"source,omitempty"`
	Status NoteStatus `json:"status,omitempty"`
}

// sharedNoteFields holds columns returned by every note query.
// Used by buildNote to avoid per-query mapping boilerplate.
type sharedNoteFields struct {
	id         pgtype.UUID
	orgID      pgtype.UUID
	userID     pgtype.UUID
	title      string
	content    *string
	para       *string
	projectID  pgtype.UUID
	tags       []string
	visibility string
	isPhi      bool
	metadata   []byte
	createdAt  pgtype.Timestamptz
	updatedAt  pgtype.Timestamptz
	archivedAt pgtype.Timestamptz
}

func buildNote(f sharedNoteFields) (*Note, error) {
	var meta noteMetadata
	if len(f.metadata) > 0 {
		if err := json.Unmarshal(f.metadata, &meta); err != nil {
			return nil, fmt.Errorf("unmarshal note metadata: %w", err)
		}
	}

	n := &Note{
		ID:         f.id.Bytes,
		OrgID:      f.orgID.Bytes,
		UserID:     f.userID.Bytes,
		Title:      f.title,
		Source:     meta.Source,
		Status:     meta.Status,
		Tags:       f.tags,
		Visibility: Visibility(f.visibility),
		IsPHI:      f.isPhi,
		CreatedAt:  f.createdAt.Time,
		UpdatedAt:  f.updatedAt.Time,
	}

	if f.content != nil {
		n.Content = *f.content
	}
	if f.para != nil {
		n.PARACategory = f.para
	}
	if f.projectID.Valid {
		id := uuid.UUID(f.projectID.Bytes)
		n.ProjectID = &id
	}
	if f.archivedAt.Valid {
		t := f.archivedAt.Time
		n.ArchivedAt = &t
	}

	return n, nil
}

func (r *pgNoteRepository) Create(ctx context.Context, orgID, userID uuid.UUID, in CreateNoteInput) (*Note, error) {
	if in.Content == "" {
		return nil, apperrors.ErrValidation.Wrap(errors.New("content is required"))
	}

	unitID, err := r.queries.FindDefaultOrgUnit(ctx, pgtype.UUID{Bytes: orgID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("find default org unit: %w", err)
	}

	source := in.Source
	if source == "" {
		source = SourceManual
	}
	visibility := in.Visibility
	if visibility == "" {
		visibility = VisibilityIndividual
	}

	meta := noteMetadata{Source: source, Status: StatusInbox}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal note metadata: %w", err)
	}

	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	content := in.Content
	row, err := r.queries.CreateNote(ctx, sqlcdb.CreateNoteParams{
		OrgID:      pgtype.UUID{Bytes: orgID, Valid: true},
		UserID:     pgtype.UUID{Bytes: userID, Valid: true},
		UnitID:     unitID,
		Title:      in.Title,
		Content:    &content,
		Tags:       tags,
		Visibility: string(visibility),
		Metadata:   metaJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	return buildNote(sharedNoteFields{
		id:         row.ID,
		orgID:      row.OrgID,
		userID:     row.UserID,
		title:      row.Title,
		content:    row.Content,
		para:       row.Para,
		projectID:  row.ProjectID,
		tags:       row.Tags,
		visibility: row.Visibility,
		isPhi:      row.IsPhi,
		metadata:   row.Metadata,
		createdAt:  row.CreatedAt,
		updatedAt:  row.UpdatedAt,
		archivedAt: row.ArchivedAt,
	})
}

//nolint:dupl // FindByID and Archive share structural shape but call different query functions.
func (r *pgNoteRepository) FindByID(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error) {
	row, err := r.queries.FindNoteByID(ctx, sqlcdb.FindNoteByIDParams{
		ID:    pgtype.UUID{Bytes: noteID, Valid: true},
		OrgID: pgtype.UUID{Bytes: orgID, Valid: true},
	})
	if err != nil {
		return nil, noteNotFound(err)
	}

	return buildNote(sharedNoteFields{
		id:         row.ID,
		orgID:      row.OrgID,
		userID:     row.UserID,
		title:      row.Title,
		content:    row.Content,
		para:       row.Para,
		projectID:  row.ProjectID,
		tags:       row.Tags,
		visibility: row.Visibility,
		isPhi:      row.IsPhi,
		metadata:   row.Metadata,
		createdAt:  row.CreatedAt,
		updatedAt:  row.UpdatedAt,
		archivedAt: row.ArchivedAt,
	})
}

//nolint:dupl // List and ListInbox share pagination shape but call different query functions.
func (r *pgNoteRepository) List(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error) {
	orgPG := pgtype.UUID{Bytes: orgID, Valid: true}
	userPG := pgtype.UUID{Bytes: userID, Valid: true}
	offset := int32((page - 1) * pageSize)

	rows, err := r.queries.ListNotes(ctx, sqlcdb.ListNotesParams{
		OrgID: orgPG, UserID: userPG, Limit: int32(pageSize), Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	total, err := r.queries.CountNotes(ctx, sqlcdb.CountNotesParams{OrgID: orgPG, UserID: userPG})
	if err != nil {
		return nil, fmt.Errorf("count notes: %w", err)
	}

	notes, err := mapNoteRows(rows, func(row sqlcdb.ListNotesRow) sharedNoteFields {
		return sharedNoteFields{
			id: row.ID, orgID: row.OrgID, userID: row.UserID,
			title: row.Title, content: row.Content, para: row.Para, projectID: row.ProjectID,
			tags: row.Tags, visibility: row.Visibility, isPhi: row.IsPhi, metadata: row.Metadata,
			createdAt: row.CreatedAt, updatedAt: row.UpdatedAt, archivedAt: row.ArchivedAt,
		}
	})
	if err != nil {
		return nil, err
	}

	return &NoteList{Notes: notes, Total: total, Page: page, PageSize: pageSize}, nil
}

//nolint:dupl // ListInbox and List share pagination shape but call different query functions.
func (r *pgNoteRepository) ListInbox(ctx context.Context, orgID, userID uuid.UUID, page, pageSize int) (*NoteList, error) {
	orgPG := pgtype.UUID{Bytes: orgID, Valid: true}
	userPG := pgtype.UUID{Bytes: userID, Valid: true}
	offset := int32((page - 1) * pageSize)

	rows, err := r.queries.ListInboxNotes(ctx, sqlcdb.ListInboxNotesParams{
		OrgID: orgPG, UserID: userPG, Limit: int32(pageSize), Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list inbox notes: %w", err)
	}

	total, err := r.queries.CountInboxNotes(ctx, sqlcdb.CountInboxNotesParams{OrgID: orgPG, UserID: userPG})
	if err != nil {
		return nil, fmt.Errorf("count inbox notes: %w", err)
	}

	notes, err := mapNoteRows(rows, func(row sqlcdb.ListInboxNotesRow) sharedNoteFields {
		return sharedNoteFields{
			id: row.ID, orgID: row.OrgID, userID: row.UserID,
			title: row.Title, content: row.Content, para: row.Para, projectID: row.ProjectID,
			tags: row.Tags, visibility: row.Visibility, isPhi: row.IsPhi, metadata: row.Metadata,
			createdAt: row.CreatedAt, updatedAt: row.UpdatedAt, archivedAt: row.ArchivedAt,
		}
	})
	if err != nil {
		return nil, err
	}

	return &NoteList{Notes: notes, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *pgNoteRepository) Update(ctx context.Context, orgID, noteID uuid.UUID, in UpdateNoteInput) (*Note, error) {
	content := in.Content
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	row, err := r.queries.UpdateNote(ctx, sqlcdb.UpdateNoteParams{
		ID:            pgtype.UUID{Bytes: noteID, Valid: true},
		OrgID:         pgtype.UUID{Bytes: orgID, Valid: true},
		Title:         in.Title,
		Content:       &content,
		Tags:          tags,
		MetadataPatch: []byte(`{}`),
	})
	if err != nil {
		return nil, noteNotFound(err)
	}

	return buildNote(sharedNoteFields{
		id:         row.ID,
		orgID:      row.OrgID,
		userID:     row.UserID,
		title:      row.Title,
		content:    row.Content,
		para:       row.Para,
		projectID:  row.ProjectID,
		tags:       row.Tags,
		visibility: row.Visibility,
		isPhi:      row.IsPhi,
		metadata:   row.Metadata,
		createdAt:  row.CreatedAt,
		updatedAt:  row.UpdatedAt,
		archivedAt: row.ArchivedAt,
	})
}

func (r *pgNoteRepository) Delete(ctx context.Context, orgID, noteID uuid.UUID) error {
	_, err := r.queries.SoftDeleteNote(ctx, sqlcdb.SoftDeleteNoteParams{
		ID:    pgtype.UUID{Bytes: noteID, Valid: true},
		OrgID: pgtype.UUID{Bytes: orgID, Valid: true},
	})
	if err != nil {
		return noteNotFound(err)
	}
	return nil
}

//nolint:dupl // Archive and FindByID share structural shape but call different query functions.
func (r *pgNoteRepository) Archive(ctx context.Context, orgID, noteID uuid.UUID) (*Note, error) {
	row, err := r.queries.ArchiveNote(ctx, sqlcdb.ArchiveNoteParams{
		ID:    pgtype.UUID{Bytes: noteID, Valid: true},
		OrgID: pgtype.UUID{Bytes: orgID, Valid: true},
	})
	if err != nil {
		return nil, noteNotFound(err)
	}

	return buildNote(sharedNoteFields{
		id:         row.ID,
		orgID:      row.OrgID,
		userID:     row.UserID,
		title:      row.Title,
		content:    row.Content,
		para:       row.Para,
		projectID:  row.ProjectID,
		tags:       row.Tags,
		visibility: row.Visibility,
		isPhi:      row.IsPhi,
		metadata:   row.Metadata,
		createdAt:  row.CreatedAt,
		updatedAt:  row.UpdatedAt,
		archivedAt: row.ArchivedAt,
	})
}

// ── helpers ────────────────────────────────────────────────────────────────────

// noteNotFound maps pgx.ErrNoRows to apperrors.ErrNotFound; wraps all other errors.
func noteNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound.Wrap(errors.New("note not found"))
	}
	return fmt.Errorf("query note: %w", err)
}

// mapNoteRows converts a slice of sqlc row types to domain Notes.
// The toFields function extracts the common sharedNoteFields from each row.
func mapNoteRows[R any](rows []R, toFields func(R) sharedNoteFields) ([]*Note, error) {
	notes := make([]*Note, 0, len(rows))
	for _, row := range rows {
		n, err := buildNote(toFields(row))
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}
