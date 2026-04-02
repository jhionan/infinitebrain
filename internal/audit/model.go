// Package audit provides operational audit recording for RBAC actions and
// resource mutations.
package audit

import (
	"time"

	"github.com/google/uuid"
)

// Entry is one immutable audit log record.
type Entry struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	ActorID    uuid.UUID
	Action     string
	TargetType string
	TargetID   *uuid.UUID
	Before     []byte // JSON
	After      []byte // JSON
	IP         string
	CreatedAt  time.Time
}
