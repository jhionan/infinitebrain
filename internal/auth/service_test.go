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

func (m *mockRepository) GetUserOrgs(_ context.Context, _ uuid.UUID) ([]auth.OrgMembership, error) {
	return []auth.OrgMembership{}, nil
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
	if _, err := svc.Register(context.Background(), "carol@test.com", "Carol", "correct-pass"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

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

func TestService_Refresh_ValidTokenReturnsNewPair(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	if _, err := svc.Register(context.Background(), "dave@test.com", "Dave", "pass12345"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

	pair, err := svc.Login(context.Background(), "dave@test.com", "pass12345")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	newPair, err := svc.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if newPair.AccessToken == pair.AccessToken {
		t.Error("Refresh should issue a new access token")
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("Refresh should issue a new refresh token (rotation)")
	}
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Error("new token pair fields must be non-empty")
	}
	if newPair.ExpiresIn <= 0 {
		t.Error("ExpiresIn must be positive")
	}
}

func TestService_Refresh_OldTokenInvalidAfterRotation(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	if _, err := svc.Register(context.Background(), "eve@test.com", "Eve", "pass12345"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

	pair, err := svc.Login(context.Background(), "eve@test.com", "pass12345")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("first Refresh: %v", err)
	}

	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for reused refresh token, got %v", err)
	}
}

func TestService_Logout_InvalidatesRefreshToken(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	if _, err := svc.Register(context.Background(), "frank@test.com", "Frank", "pass12345"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}

	pair, err := svc.Login(context.Background(), "frank@test.com", "pass12345")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized after logout, got %v", err)
	}
}

func TestService_Logout_AlreadyExpiredTokenIsNoOp(t *testing.T) {
	svc := newTestService(newMockRepo())
	err := svc.Logout(context.Background(), "nonexistent-refresh-token")
	if err != nil {
		t.Errorf("Logout of nonexistent token should be no-op, got: %v", err)
	}
}

func TestService_GetUserOrgs_ReturnsForbiddenForWrongUser(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	// Build a context with claims for userA.
	signerA := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	userA := uuid.New()
	token, err := signerA.Sign(&auth.User{ID: userA, OrgID: uuid.New(), Email: "a@a.com", Role: "editor"})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := signerA.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	ctx := auth.ContextWithClaims(context.Background(), claims)

	// Try to query a different user's orgs.
	userB := uuid.New()
	_, err = svc.GetUserOrgs(ctx, userB)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestService_Me_ReturnsUserProfile(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)
	pair, err := svc.Register(context.Background(), "grace@test.com", "Grace", "pass12345")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Extract user ID from access token
	signer := auth.NewSigner("test-secret-that-is-32chars-long!!", 15*time.Minute)
	claims, err := signer.Verify(pair.AccessToken)
	if err != nil {
		t.Fatalf("verifying token: %v", err)
	}

	profile, err := svc.Me(context.Background(), claims.UserID.String())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if profile.Email != "grace@test.com" {
		t.Errorf("Email = %q, want grace@test.com", profile.Email)
	}
	if profile.DisplayName != "Grace" {
		t.Errorf("DisplayName = %q, want Grace", profile.DisplayName)
	}
}

func TestService_Me_InvalidUserIDReturnsValidation(t *testing.T) {
	svc := newTestService(newMockRepo())
	_, err := svc.Me(context.Background(), "not-a-uuid")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_Register_ShortPasswordReturnsValidation(t *testing.T) {
	svc := newTestService(newMockRepo())
	_, err := svc.Register(context.Background(), "short@test.com", "Short", "abc")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation for short password, got %v", err)
	}
}

func TestService_Register_EmptyFieldsReturnsValidation(t *testing.T) {
	svc := newTestService(newMockRepo())
	tests := []struct {
		name     string
		email    string
		dispName string
		password string
	}{
		{"empty email", "", "Alice", "pass12345"},
		{"empty displayName", "a@b.com", "", "pass12345"},
		{"empty password", "a@b.com", "Alice", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.email, tt.dispName, tt.password)
			if !errors.Is(err, apperrors.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestService_GetUserOrgs_SameUserReturnsOrgs(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	userID := uuid.New()
	token, err := signer.Sign(&auth.User{ID: userID, OrgID: uuid.New(), Email: "h@h.com", Role: "owner"})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	ctx := auth.ContextWithClaims(context.Background(), claims)

	orgs, err := svc.GetUserOrgs(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserOrgs: %v", err)
	}
	if orgs == nil {
		t.Error("expected non-nil orgs slice")
	}
}

func TestService_Refresh_InvalidTokenReturnsUnauthorized(t *testing.T) {
	svc := newTestService(newMockRepo())
	_, err := svc.Refresh(context.Background(), "bogus-refresh-token")
	if !errors.Is(err, apperrors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// mockRepoFailCreateSession wraps mockRepository but fails on CreateSession.
type mockRepoFailCreateSession struct {
	mockRepository
}

func (m *mockRepoFailCreateSession) CreateSession(_ context.Context, _ *auth.Session) (*auth.Session, error) {
	return nil, errors.New("db connection refused")
}

func TestService_Register_CreateSessionFailureReturnsError(t *testing.T) {
	repo := &mockRepoFailCreateSession{mockRepository: *newMockRepo()}
	svc := newTestService(repo)
	_, err := svc.Register(context.Background(), "failsession@test.com", "Fail", "pass12345")
	if err == nil {
		t.Fatal("expected error when CreateSession fails, got nil")
	}
}

func TestService_Login_CreateSessionFailureReturnsError(t *testing.T) {
	repo := &mockRepoFailCreateSession{mockRepository: *newMockRepo()}
	// Pre-seed the user directly via the embedded mockRepository.
	svc := newTestService(&repo.mockRepository)
	if _, err := svc.Register(context.Background(), "faillogin@test.com", "Fail", "pass12345"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}
	// Now swap to the failing repo.
	svc2 := newTestService(repo)
	_, err := svc2.Login(context.Background(), "faillogin@test.com", "pass12345")
	if err == nil {
		t.Fatal("expected error when CreateSession fails, got nil")
	}
}
