package auth

import "context"

// Service is the business logic contract for authentication.
type Service interface {
	Register(ctx context.Context, email, displayName, password string) (*TokenPair, error)
	Login(ctx context.Context, email, password string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID string) (*UserProfile, error)
}
