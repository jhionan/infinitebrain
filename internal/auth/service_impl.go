package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type serviceImpl struct {
	repo            Repository
	signer          *Signer
	pepper          string
	refreshDuration time.Duration
}

// NewService creates an AuthService.
func NewService(repo Repository, signer *Signer, pepper string, refreshDuration time.Duration) Service {
	return &serviceImpl{
		repo:            repo,
		signer:          signer,
		pepper:          pepper,
		refreshDuration: refreshDuration,
	}
}

// Register creates a new user account and returns a token pair.
// A personal org is created atomically with the user.
func (s *serviceImpl) Register(ctx context.Context, email, displayName, password string) (*TokenPair, error) {
	if email == "" || displayName == "" || password == "" {
		return nil, apperrors.ErrValidation.Wrap(fmt.Errorf("email, displayName, and password are required"))
	}
	if len(password) < 8 {
		return nil, apperrors.ErrValidation.Wrap(fmt.Errorf("password must be at least 8 characters"))
	}

	hash, err := HashPassword(password, s.pepper)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.repo.Register(ctx, email, displayName, hash, 1)
	if err != nil {
		return nil, fmt.Errorf("registering user: %w", err)
	}

	return s.issueTokenPair(ctx, user)
}

// Login validates credentials and issues a token pair.
// Returns ErrUnauthorized for both unknown email and wrong password (no enumeration).
func (s *serviceImpl) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("invalid credentials"))
		}
		return nil, fmt.Errorf("looking up user by email: %w", err)
	}

	if !VerifyPassword(password, s.pepper, user.PasswordHash) {
		return nil, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("invalid credentials"))
	}

	return s.issueTokenPair(ctx, user)
}

// Refresh validates the refresh token and issues a new token pair (rotation).
func (s *serviceImpl) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashRefreshToken(refreshToken)

	session, err := s.repo.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrUnauthorized.Wrap(fmt.Errorf("invalid or expired refresh token"))
		}
		return nil, fmt.Errorf("looking up session: %w", err)
	}

	// Rotate: invalidate old session immediately.
	if err := s.repo.DeleteSession(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("revoking old session: %w", err)
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("loading user for refresh: %w", err)
	}

	return s.issueTokenPair(ctx, user)
}

// Logout invalidates the refresh token.
func (s *serviceImpl) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashRefreshToken(refreshToken)
	session, err := s.repo.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil // already gone — treat as success
	}
	if err := s.repo.DeleteSession(ctx, session.ID); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// Me returns the profile for the authenticated user.
func (s *serviceImpl) Me(ctx context.Context, userID string) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperrors.ErrValidation.Wrap(fmt.Errorf("invalid user ID"))
	}
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading user profile: %w", err)
	}
	return &UserProfile{
		ID:          user.ID,
		OrgID:       user.OrgID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
	}, nil
}

// GetUserOrgs returns all orgs the user belongs to.
// The caller must be the same user as userID — querying another user's orgs is forbidden.
func (s *serviceImpl) GetUserOrgs(ctx context.Context, userID uuid.UUID) ([]OrgMembership, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.UserID != userID {
		return nil, apperrors.ErrForbidden.Wrap(fmt.Errorf("cannot query another user's orgs"))
	}
	orgs, err := s.repo.GetUserOrgs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orgs: %w", err)
	}
	return orgs, nil
}

// issueTokenPair creates a new access token + refresh token session.
func (s *serviceImpl) issueTokenPair(ctx context.Context, user *User) (*TokenPair, error) {
	accessToken, err := s.signer.Sign(user)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	rawRefresh, err := newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	_, err = s.repo.CreateSession(ctx, &Session{
		UserID:    user.ID,
		OrgID:     user.OrgID,
		TokenHash: hashRefreshToken(rawRefresh),
		ExpiresAt: time.Now().Add(s.refreshDuration),
	})
	if err != nil {
		return nil, fmt.Errorf("persisting session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.signer.Duration().Seconds()),
	}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
