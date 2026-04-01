package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rian/infinite_brain/internal/auth"
)

func makeTestSigner() *auth.Signer {
	return auth.NewSigner("test-secret-that-is-32chars-long!!", 15*time.Minute)
}

func makeValidToken(t *testing.T) string {
	t.Helper()
	signer := makeTestSigner()
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "test@example.com",
		Role:  "owner",
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return token
}

func TestAuth_ValidToken_CallsNext(t *testing.T) {
	signer := makeTestSigner()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
		}
		if claims.Email != "test@example.com" {
			t.Errorf("Email = %q, want test@example.com", claims.Email)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.Auth(signer)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+makeValidToken(t))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuth_MissingToken_Returns401(t *testing.T) {
	signer := makeTestSigner()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("next should not be called without token")
	})

	handler := auth.Auth(signer)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	signer := makeTestSigner()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("next should not be called with invalid token")
	})

	handler := auth.Auth(signer)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestClaimsFromContext_MissingReturnsNotOk(t *testing.T) {
	_, ok := auth.ClaimsFromContext(context.Background())
	if ok {
		t.Error("expected false for context without claims")
	}
}
