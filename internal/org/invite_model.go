package org

import (
	"time"

	"github.com/google/uuid"
)

// Invite is a pending invitation to join an org.
type Invite struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Email      string
	Role       string
	InvitedBy  uuid.UUID
	Token      string
	ExpiresAt  time.Time
	AcceptedAt *time.Time
	CreatedAt  time.Time
}
