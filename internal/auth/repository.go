package auth

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the data access contract for the auth domain.
// Interface defined at the consumption point (CLAUDE.md rule).
type Repository interface {
	// Register atomically creates a personal org and user in one transaction.
	Register(ctx context.Context, email, displayName, passwordHash string, pepperVersion int16) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	CreateSession(ctx context.Context, s *Session) (*Session, error)
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	// GetUserOrgs returns all orgs the user belongs to with their role in each.
	GetUserOrgs(ctx context.Context, userID uuid.UUID) ([]OrgMembership, error)
}
