package org

import (
	"context"

	"github.com/google/uuid"
)

// InviteRepository is the data access contract for org invitations.
type InviteRepository interface {
	Create(ctx context.Context, i *Invite) (*Invite, error)
	FindByToken(ctx context.Context, token string) (*Invite, error)
	Accept(ctx context.Context, id uuid.UUID) error
}
