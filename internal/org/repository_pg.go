package org

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	sqlcdb "github.com/rian/infinite_brain/db/sqlc"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcdb.Queries
}

// NewRepository returns a PostgreSQL-backed org Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool, queries: sqlcdb.New(pool)}
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*Org, error) {
	row, err := r.queries.FindOrgByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapNotFound(err, "org not found")
	}
	return mapOrgFromFindByID(row)
}

func (r *pgRepository) FindBySlug(ctx context.Context, slug string) (*Org, error) {
	row, err := r.queries.FindOrgBySlug(ctx, slug)
	if err != nil {
		return nil, mapNotFound(err, "org not found")
	}
	return mapOrgFromFindBySlug(row)
}

func (r *pgRepository) Update(ctx context.Context, id uuid.UUID, name string, settings OrgSettings) (*Org, error) {
	settingsJSON, err := MarshalSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}
	row, err := r.queries.UpdateOrg(ctx, sqlcdb.UpdateOrgParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		Name:     name,
		Settings: settingsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("update org: %w", err)
	}
	return mapOrgFromUpdateRow(row)
}

func (r *pgRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteOrg(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (r *pgRepository) AddMember(ctx context.Context, orgID, userID uuid.UUID, role string, invitedBy *uuid.UUID) error {
	var invBy pgtype.UUID
	if invitedBy != nil {
		invBy = pgtype.UUID{Bytes: *invitedBy, Valid: true}
	}
	return r.queries.AddOrgMember(ctx, sqlcdb.AddOrgMemberParams{
		OrgID:     pgtype.UUID{Bytes: orgID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Role:      role,
		InvitedBy: invBy,
	})
}

func (r *pgRepository) FindMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error) {
	row, err := r.queries.FindOrgMember(ctx, sqlcdb.FindOrgMemberParams{
		OrgID:  pgtype.UUID{Bytes: orgID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return nil, mapNotFound(err, "member not found")
	}
	return mapMemberFromOrgMember(row), nil
}

func (r *pgRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error) {
	rows, err := r.queries.ListOrgMembers(ctx, pgtype.UUID{Bytes: orgID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list org members: %w", err)
	}
	members := make([]Member, len(rows))
	for i, row := range rows {
		m := mapMemberFromListRow(row)
		members[i] = *m
	}
	return members, nil
}

func (r *pgRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	return r.queries.UpdateOrgMemberRole(ctx, sqlcdb.UpdateOrgMemberRoleParams{
		OrgID:  pgtype.UUID{Bytes: orgID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Role:   role,
	})
}

func (r *pgRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return r.queries.RemoveOrgMember(ctx, sqlcdb.RemoveOrgMemberParams{
		OrgID:  pgtype.UUID{Bytes: orgID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
}

func (r *pgRepository) CountMembers(ctx context.Context, orgID uuid.UUID) (int64, error) {
	n, err := r.queries.CountOrgMembers(ctx, pgtype.UUID{Bytes: orgID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("count org members: %w", err)
	}
	return n, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func mapOrgFields(id pgtype.UUID, name, slug, plan string, maxMembers *int32, settingsJSON []byte, phiEnabled bool, createdAt, updatedAt pgtype.Timestamptz) (*Org, error) {
	var settings OrgSettings
	if err := UnmarshalSettings(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("unmarshal org settings: %w", err)
	}
	o := &Org{
		ID:         id.Bytes,
		Name:       name,
		Slug:       slug,
		Plan:       plan,
		PhiEnabled: phiEnabled,
		Settings:   settings,
		CreatedAt:  createdAt.Time,
		UpdatedAt:  updatedAt.Time,
	}
	if maxMembers != nil {
		v := int(*maxMembers)
		o.MaxMembers = &v
	}
	return o, nil
}

func mapOrgFromFindByID(row sqlcdb.FindOrgByIDRow) (*Org, error) {
	return mapOrgFields(row.ID, row.Name, row.Slug, row.Plan, row.MaxMembers, row.Settings, row.PhiEnabled, row.CreatedAt, row.UpdatedAt)
}

func mapOrgFromFindBySlug(row sqlcdb.FindOrgBySlugRow) (*Org, error) {
	return mapOrgFields(row.ID, row.Name, row.Slug, row.Plan, row.MaxMembers, row.Settings, row.PhiEnabled, row.CreatedAt, row.UpdatedAt)
}

func mapOrgFromUpdateRow(row sqlcdb.UpdateOrgRow) (*Org, error) {
	return mapOrgFields(row.ID, row.Name, row.Slug, row.Plan, row.MaxMembers, row.Settings, row.PhiEnabled, row.CreatedAt, row.UpdatedAt)
}

func mapMemberFromOrgMember(row sqlcdb.OrgMember) *Member {
	m := &Member{
		OrgID:    row.OrgID.Bytes,
		UserID:   row.UserID.Bytes,
		Role:     row.Role,
		JoinedAt: row.JoinedAt.Time,
	}
	if row.InvitedBy.Valid {
		id := uuid.UUID(row.InvitedBy.Bytes)
		m.InvitedBy = &id
	}
	return m
}

func mapMemberFromListRow(row sqlcdb.ListOrgMembersRow) *Member {
	m := &Member{
		OrgID:       row.OrgID.Bytes,
		UserID:      row.UserID.Bytes,
		Role:        row.Role,
		JoinedAt:    row.JoinedAt.Time,
		Email:       row.Email,
		DisplayName: row.DisplayName,
	}
	if row.InvitedBy.Valid {
		id := uuid.UUID(row.InvitedBy.Bytes)
		m.InvitedBy = &id
	}
	return m
}

func mapNotFound(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound.Wrap(errors.New(msg))
	}
	return err
}
