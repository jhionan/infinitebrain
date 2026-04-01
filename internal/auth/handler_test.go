package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
)

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
