// Package auth implements JWT-based authentication for Infinite Brain.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are the JWT payload for every access token.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	OrgID  uuid.UUID `json:"org_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// Signer creates and verifies HS256 JWT access tokens.
type Signer struct {
	secret   []byte
	duration time.Duration
}

// NewSigner creates a Signer. secret must be ≥ 32 bytes; duration is token lifetime.
func NewSigner(secret string, duration time.Duration) *Signer {
	return &Signer{secret: []byte(secret), duration: duration}
}

// Sign issues a signed JWT for the given user.
func (s *Signer) Sign(user *User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		OrgID:  user.OrgID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.duration)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("signing jwt: %w", err)
	}
	return token, nil
}

// Verify parses and validates a JWT, returning the embedded claims.
func (s *Signer) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid jwt claims")
	}
	return claims, nil
}

// Duration returns the configured token lifetime (used by service for ExpiresIn).
func (s *Signer) Duration() time.Duration { return s.duration }
