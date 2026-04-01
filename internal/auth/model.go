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

// Session represents an active refresh token.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	OrgID     uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// TokenPair is returned to the client on successful login or refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token lifetime in seconds
}

// UserProfile is the safe public view returned from GET /auth/me.
type UserProfile struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}
