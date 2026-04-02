package org

import (
	"context"

	"github.com/google/uuid"
)

// InviteService manages the invite lifecycle: create and accept.
type InviteService interface {
	// CreateInvite issues a new invite for email to join orgID with role.
	// callerID must have PermManageMembers in the org.
	CreateInvite(ctx context.Context, orgID uuid.UUID, email, role string, callerID uuid.UUID) (*Invite, error)

	// AcceptInvite accepts an invite by token. The accepting user (userID) is added
	// to org_members with the invite's role. Returns ErrNotFound if the token is
	// invalid or expired.
	AcceptInvite(ctx context.Context, token string, userID uuid.UUID) error
}
