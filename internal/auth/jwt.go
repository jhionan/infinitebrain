// Package auth implements JWT generation and validation for the Infinite Brain API.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	// TokenTypeAccess is a short-lived JWT used to authenticate API requests.
	TokenTypeAccess TokenType = "access"
)

// Claims are the verified contents of an access token.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	OrgID  uuid.UUID `json:"org_id"`
	Email  string    `json:"email"`
	JTI    uuid.UUID `json:"jti"`
}

// jwtClaims is the internal JWT payload used with golang-jwt.
type jwtClaims struct {
	jwt.RegisteredClaims
	UserID string    `json:"user_id"`
	OrgID  string    `json:"org_id"`
	Email  string    `json:"email"`
	Type   TokenType `json:"type"`
}

// Signer generates and validates JWTs.
// It is safe for concurrent use.
type Signer struct {
	secret   []byte
	duration time.Duration
}

// NewSigner creates a Signer using the provided HMAC-SHA256 secret and token duration.
// secret must be at least 32 bytes; NewSigner panics if the invariant is violated
// (this is a startup-time configuration error, not a runtime error).
func NewSigner(secret string, accessDuration time.Duration) *Signer {
	if len(secret) < 32 {
		panic("auth: JWT secret must be at least 32 characters")
	}
	return &Signer{
		secret:   []byte(secret),
		duration: accessDuration,
	}
}

// Issue generates a signed access token for the given user.
// ctx is accepted for future tracing / audit integration.
func (s *Signer) Issue(_ context.Context, userID, orgID uuid.UUID, email string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.duration)),
		},
		UserID: userID.String(),
		OrgID:  orgID.String(),
		Email:  email,
		Type:   TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("signing jwt: %w", err)
	}
	return signed, nil
}

// Validate parses and verifies a signed access token, returning the embedded claims.
// It returns ErrTokenInvalid for any malformed or expired token.
func (s *Signer) Validate(_ context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	if c.Type != TokenTypeAccess {
		return nil, fmt.Errorf("%w: wrong token type %q", ErrTokenInvalid, c.Type)
	}

	userID, err := uuid.Parse(c.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id claim", ErrTokenInvalid)
	}
	orgID, err := uuid.Parse(c.OrgID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid org_id claim", ErrTokenInvalid)
	}
	jti, err := uuid.Parse(c.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid jti claim", ErrTokenInvalid)
	}

	return &Claims{
		UserID: userID,
		OrgID:  orgID,
		Email:  c.Email,
		JTI:    jti,
	}, nil
}

// ErrTokenInvalid is returned when a token cannot be verified or has expired.
var ErrTokenInvalid = errors.New("invalid or expired token")
