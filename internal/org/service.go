package org

import (
	"context"

	"github.com/google/uuid"
)

// Service is the org business logic contract.
type Service interface {
	Get(ctx context.Context, slug string) (*Org, error)
	Update(ctx context.Context, orgID uuid.UUID, callerID uuid.UUID, name string, settings OrgSettings) (*Org, error)
	Delete(ctx context.Context, orgID uuid.UUID, callerID uuid.UUID) error

	ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	AddMember(ctx context.Context, orgID, userID uuid.UUID, role string, inviterID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, targetUserID, callerID uuid.UUID, role string) error
	RemoveMember(ctx context.Context, orgID, targetUserID, callerID uuid.UUID) error
}
