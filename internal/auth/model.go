package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is the auth domain model.
type User struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	Email         string
	DisplayName   string
	Role          string
	PasswordHash  string
	PepperVersion int16
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
