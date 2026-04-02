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

type pgInviteRepository struct {
	queries *sqlcdb.Queries
}

// NewInviteRepository returns a PostgreSQL-backed InviteRepository.
func NewInviteRepository(pool *pgxpool.Pool) InviteRepository {
	return &pgInviteRepository{queries: sqlcdb.New(pool)}
}

func (r *pgInviteRepository) Create(ctx context.Context, i *Invite) (*Invite, error) {
	row, err := r.queries.CreateOrgInvite(ctx, sqlcdb.CreateOrgInviteParams{
		OrgID:     pgtype.UUID{Bytes: i.OrgID, Valid: true},
		Email:     i.Email,
		Role:      i.Role,
		InvitedBy: pgtype.UUID{Bytes: i.InvitedBy, Valid: true},
		Token:     i.Token,
	})
	if err != nil {
		return nil, fmt.Errorf("create org invite: %w", err)
	}
	return mapInviteFromRow(row), nil
}

func (r *pgInviteRepository) FindByToken(ctx context.Context, token string) (*Invite, error) {
	row, err := r.queries.FindOrgInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound.Wrap(errors.New("invite not found or expired"))
		}
		return nil, fmt.Errorf("find org invite: %w", err)
	}
	return mapInviteFromRow(row), nil
}

func (r *pgInviteRepository) Accept(ctx context.Context, id uuid.UUID) error {
	_, err := r.queries.AcceptOrgInvite(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrConflict.Wrap(fmt.Errorf("invite already accepted"))
		}
		return fmt.Errorf("accept org invite: %w", err)
	}
	return nil
}

func mapInviteFromRow(row sqlcdb.OrgInvite) *Invite {
	inv := &Invite{
		ID:        row.ID.Bytes,
		OrgID:     row.OrgID.Bytes,
		Email:     row.Email,
		Role:      row.Role,
		InvitedBy: row.InvitedBy.Bytes,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.AcceptedAt.Valid {
		t := row.AcceptedAt.Time
		inv.AcceptedAt = &t
	}
	return inv
}
