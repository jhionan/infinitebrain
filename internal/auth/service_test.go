package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// mockRepository is a test double for auth.Repository.
type mockRepository struct {
	users    map[string]*auth.User
	sessions map[string]*auth.Session
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		users:    make(map[string]*auth.User),
		sessions: make(map[string]*auth.Session),
	}
}

func (m *mockRepository) Register(_ context.Context, email, displayName, passwordHash string, pepperVersion int16) (*auth.User, error) {
	if _, exists := m.users[email]; exists {
		return nil, apperrors.ErrConflict.Wrap(errors.New("email already registered"))
	}
	u := &auth.User{
		ID:            uuid.New(),
		OrgID:         uuid.New(),
		Email:         email,
		DisplayName:   displayName,
		Role:          "owner",
		PasswordHash:  passwordHash,
		PepperVersion: pepperVersion,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.users[email] = u
	return u, nil
}

func (m *mockRepository) FindUserByEmail(_ context.Context, email string) (*auth.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("user not found"))
	}
	return u, nil
}

func (m *mockRepository) FindUserByID(_ context.Context, id uuid.UUID) (*auth.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, apperrors.ErrNotFound.Wrap(errors.New("user not found"))
}

func (m *mockRepository) CreateSession(_ context.Context, s *auth.Session) (*auth.Session, error) {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	m.sessions[s.TokenHash] = s
	return s, nil
}

func (m *mockRepository) FindSessionByTokenHash(_ context.Context, tokenHash string) (*auth.Session, error) {
	s, ok := m.sessions[tokenHash]
	if !ok || time.Now().After(s.ExpiresAt) {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("session not found"))
	}
	return s, nil
}

func (m *mockRepository) DeleteSession(_ context.Context, id uuid.UUID) error {
	for k, s := range m.sessions {
		if s.ID == id {
			delete(m.sessions, k)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) DeleteSessionsByUserID(_ context.Context, userID uuid.UUID) error {
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func newTestService(repo auth.Repository) auth.Service {
	signer := auth.NewSigner("test-secret-that-is-32chars-long!!", 15*time.Minute)
	return auth.NewService(repo, signer, "test-pepper", 7*24*time.Hour)
}

func TestService_Register_ReturnsTokenPair(t *testing.T) {
	svc := newTestService(newMockRepo())
	pair, err := svc.Register(context.Background(), "alice@test.com", "Alice", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("TokenPair fields must be non-empty")
	}
	if pair.ExpiresIn <= 0 {
		t.Error("ExpiresIn must be positive")
	}
}

func TestService_Register_DuplicateEmailReturnsConflict(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	if _, err := svc.Register(context.Background(), "a@b.com", "A", "pass12345"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(context.Background(), "a@b.com", "A2", "pass12345")
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_Login_ValidCredentialsReturnsTokenPair(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	if _, err := svc.Register(context.Background(), "bob@test.com", "Bob", "secret123"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	pair, err := svc.Login(context.Background(), "bob@test.com", "secret123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestService_Login_WrongPasswordReturnsUnauthorized(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	svc.Register(context.Background(), "carol@test.com", "Carol", "correct-pass") //nolint:errcheck

	_, err := svc.Login(context.Background(), "carol@test.com", "wrong-pass")
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_Login_UnknownEmailReturnsUnauthorized(t *testing.T) {
	svc := newTestService(newMockRepo())
	_, err := svc.Login(context.Background(), "nobody@test.com", "pass")
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
