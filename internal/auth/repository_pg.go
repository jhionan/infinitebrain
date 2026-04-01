package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	sqlcdb "github.com/rian/infinite_brain/db/sqlc"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcdb.Queries
}

// NewRepository returns a PostgreSQL-backed auth Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool, queries: sqlcdb.New(pool)}
}

func (r *pgRepository) Register(ctx context.Context, email, displayName, passwordHash string, pepperVersion int16) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin register tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := sqlcdb.New(tx)

	org, err := qtx.CreateOrg(ctx, sqlcdb.CreateOrgParams{
		Name: displayName + "'s Brain",
		Slug: slugifyEmail(email),
		Plan: "personal",
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.ErrConflict.Wrap(fmt.Errorf("email already registered: %s", email))
		}
		return nil, fmt.Errorf("create org: %w", err)
	}

	row, err := qtx.CreateUser(ctx, sqlcdb.CreateUserParams{
		OrgID:         org.ID,
		Email:         email,
		DisplayName:   displayName,
		Role:          "owner",
		PasswordHash:  &passwordHash,
		PepperVersion: pepperVersion,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.ErrConflict.Wrap(fmt.Errorf("email already registered: %s", email))
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit register tx: %w", err)
	}

	return mapCreateUserRow(row), nil
}

func (r *pgRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, mapNotFound(err, "user not found")
	}
	return mapFindUserByEmailRow(row), nil
}

func (r *pgRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.queries.FindUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapNotFound(err, "user not found")
	}
	return mapFindUserByIDRow(row), nil
}

func (r *pgRepository) CreateSession(ctx context.Context, s *Session) (*Session, error) {
	row, err := r.queries.CreateSession(ctx, sqlcdb.CreateSessionParams{
		UserID:    pgtype.UUID{Bytes: s.UserID, Valid: true},
		OrgID:     pgtype.UUID{Bytes: s.OrgID, Valid: true},
		TokenHash: s.TokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: s.ExpiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &Session{
		ID:        row.ID.Bytes,
		UserID:    row.UserID.Bytes,
		OrgID:     row.OrgID.Bytes,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *pgRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	row, err := r.queries.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, mapNotFound(err, "session not found or expired")
	}
	return &Session{
		ID:        row.ID.Bytes,
		UserID:    row.UserID.Bytes,
		OrgID:     row.OrgID.Bytes,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *pgRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteSession(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (r *pgRepository) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.queries.DeleteSessionsByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func mapCreateUserRow(row sqlcdb.CreateUserRow) *User {
	hash := ""
	if row.PasswordHash != nil {
		hash = *row.PasswordHash
	}
	return &User{
		ID:            row.ID.Bytes,
		OrgID:         row.OrgID.Bytes,
		Email:         row.Email,
		DisplayName:   row.DisplayName,
		Role:          row.Role,
		PasswordHash:  hash,
		PepperVersion: row.PepperVersion,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func mapFindUserByEmailRow(r sqlcdb.FindUserByEmailRow) *User {
	hash := ""
	if r.PasswordHash != nil {
		hash = *r.PasswordHash
	}
	return &User{
		ID:            r.ID.Bytes,
		OrgID:         r.OrgID.Bytes,
		Email:         r.Email,
		DisplayName:   r.DisplayName,
		Role:          r.Role,
		PasswordHash:  hash,
		PepperVersion: r.PepperVersion,
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}
}

func mapFindUserByIDRow(r sqlcdb.FindUserByIDRow) *User {
	hash := ""
	if r.PasswordHash != nil {
		hash = *r.PasswordHash
	}
	return &User{
		ID:            r.ID.Bytes,
		OrgID:         r.OrgID.Bytes,
		Email:         r.Email,
		DisplayName:   r.DisplayName,
		Role:          r.Role,
		PasswordHash:  hash,
		PepperVersion: r.PepperVersion,
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}
}

// slugifyEmail converts an email to a URL-safe org slug.
// rian@example.com → rian-at-example-com
func slugifyEmail(email string) string {
	s := strings.ToLower(email)
	s = strings.ReplaceAll(s, "@", "-at-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "+", "-")
	return s
}

// isUniqueViolation checks for PostgreSQL unique constraint violation (code 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// mapNotFound maps pgx.ErrNoRows → apperrors.ErrNotFound; passes other errors through.
func mapNotFound(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound.Wrap(errors.New(msg))
	}
	return err
}
