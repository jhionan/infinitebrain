package org

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the data access contract for org management.
type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Org, error)
	FindBySlug(ctx context.Context, slug string) (*Org, error)
	Update(ctx context.Context, id uuid.UUID, name string, settings OrgSettings) (*Org, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// AddMember adds userID to orgID with the given role.
	// Idempotent: if the membership already exists, this is a no-op (no error, no update).
	// Callers that need to distinguish "added" from "already a member" must call FindMember first.
	AddMember(ctx context.Context, orgID, userID uuid.UUID, role string, invitedBy *uuid.UUID) error
	// FindMember returns the membership for the given org+user pair.
	// Email and DisplayName are NOT populated — use ListMembers to get those fields.
	FindMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error)
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	CountMembers(ctx context.Context, orgID uuid.UUID) (int64, error)
}
