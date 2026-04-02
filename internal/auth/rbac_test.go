package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
)

func TestRequire_AllowsRequestWithSufficientRole(t *testing.T) {
	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "alice@example.com",
		Role:  "admin",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := auth.Auth(signer)(auth.Require(auth.PermManageMembers)(next))

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequire_Returns403ForInsufficientRole(t *testing.T) {
	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "viewer@example.com",
		Role:  "viewer",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := auth.Auth(signer)(auth.Require(auth.PermManageMembers)(next))

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body)
	}
}

func TestRequire_Returns401WhenNoClaims(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Require applied WITHOUT Auth middleware — no claims in context.
	handler := auth.Require(auth.PermManageMembers)(next)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequire_ErrorBodyContainsRoleAndPermission(t *testing.T) {
	signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
	token, err := signer.Sign(&auth.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Email: "viewer@example.com",
		Role:  "viewer",
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
	handler := auth.Auth(signer)(auth.Require(auth.PermBilling)(next))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "viewer") || !strings.Contains(body, string(auth.PermBilling)) {
		t.Errorf("expected body to contain role and perm, got: %s", body)
	}
}
