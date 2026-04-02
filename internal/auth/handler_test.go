package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
)

// mockService is a minimal Service implementation for handler unit tests.
// It delegates nothing — tests that need real logic use newTestService(newMockRepo()) instead.
type mockService struct{}

func (m *mockService) Register(_ context.Context, _, _, _ string) (*auth.TokenPair, error) {
	return nil, nil
}

func (m *mockService) Login(_ context.Context, _, _ string) (*auth.TokenPair, error) {
	return nil, nil
}

func (m *mockService) Refresh(_ context.Context, _ string) (*auth.TokenPair, error) {
	return nil, nil
}

func (m *mockService) Logout(_ context.Context, _ string) error {
	return nil
}

func (m *mockService) Me(_ context.Context, _ string) (*auth.UserProfile, error) {
	return &auth.UserProfile{}, nil
}

func (m *mockService) GetUserOrgs(_ context.Context, _ uuid.UUID) ([]auth.OrgMembership, error) {
	return []auth.OrgMembership{}, nil
}

func newTestHandler() *auth.Handler {
	svc := newTestService(newMockRepo())
	return auth.NewHandler(svc, zerolog.Nop())
}

func TestHandler_Register_ValidBody_Returns201(t *testing.T) {
	h := newTestHandler()
	body := `{"email":"test@example.com","display_name":"Test User","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	data := resp["data"].(map[string]any)
	if data["access_token"] == "" {
		t.Error("expected non-empty access_token in response")
	}
}

func TestHandler_Register_MissingEmail_Returns422(t *testing.T) {
	h := newTestHandler()
	body := `{"display_name":"Test","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestHandler_Login_ValidCredentials_Returns200(t *testing.T) {
	svc := newTestService(newMockRepo())
	if _, err := svc.Register(context.Background(), "user@example.com", "User", "password123"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}
	h := auth.NewHandler(svc, zerolog.Nop())

	body := `{"email":"user@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Login_WrongPassword_Returns401(t *testing.T) {
	svc := newTestService(newMockRepo())
	if _, err := svc.Register(context.Background(), "user2@example.com", "User2", "correct12"); err != nil {
		t.Fatalf("setup Register: %v", err)
	}
	h := auth.NewHandler(svc, zerolog.Nop())

	body := `{"email":"user2@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandler_Me_WithoutAuth_Returns401(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	h.Me(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthHandler_MyOrgs_Returns200WithClaims(t *testing.T) {
	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "user@example.com",
		Role:  "owner",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	handler := auth.NewHandler(&mockService{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	auth.Auth(signer)(http.HandlerFunc(handler.MyOrgs)).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestAuthHandler_MyOrgs_Returns401WithNoAuth(t *testing.T) {
	handler := auth.NewHandler(&mockService{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/orgs", nil)
	rr := httptest.NewRecorder()
	handler.MyOrgs(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthHandler_MyPermissions_Returns200WithClaims(t *testing.T) {
	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "user@example.com",
		Role:  "editor",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	handler := auth.NewHandler(&mockService{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	auth.Auth(signer)(http.HandlerFunc(handler.MyPermissions)).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}
