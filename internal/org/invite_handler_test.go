package org_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/org"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

type mockInviteSvc struct {
	createErr error
	acceptErr error
}

func (m *mockInviteSvc) CreateInvite(_ context.Context, _ uuid.UUID, email, role string, _ uuid.UUID) (*org.Invite, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &org.Invite{
		ID:        uuid.New(),
		Email:     email,
		Role:      role,
		Token:     "generated-token",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockInviteSvc) AcceptInvite(_ context.Context, token string, _ uuid.UUID) error {
	if m.acceptErr != nil {
		return m.acceptErr
	}
	if token == "bad-token" {
		return apperrors.ErrNotFound.Wrap(errors.New("invite not found"))
	}
	return nil
}

// makeSignedRequest creates an HTTP request with JWT claims injected into context.
func makeSignedRequest(t *testing.T, method, path string, body any, role string) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	if role != "" {
		signer := auth.NewSigner("supersecretjwtkey12345678901234567890", time.Hour)
		token, err := signer.Sign(&auth.User{
			ID:    uuid.New(),
			OrgID: uuid.New(),
			Email: "user@example.com",
			Role:  role,
		})
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		claims, err := signer.Verify(token)
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
		ctx := auth.ContextWithClaims(req.Context(), claims)
		req = req.WithContext(ctx)
	}
	return req
}

func withOrgContext(req *http.Request, o *org.Org) *http.Request {
	ctx := context.WithValue(req.Context(), org.OrgContextKey{}, o)
	return req.WithContext(ctx)
}

func TestInviteHandler_CreateInvite_Returns201(t *testing.T) {
	handler := org.NewInviteHandler(&mockInviteSvc{}, zerolog.Nop())
	o := &org.Org{ID: uuid.New(), Slug: "test-org", Plan: "teams"}

	req := makeSignedRequest(t, http.MethodPost, "/api/v1/orgs/test-org/invites",
		map[string]string{"email": "new@example.com", "role": "editor"}, "admin")
	req = withOrgContext(req, o)
	rr := httptest.NewRecorder()

	handler.CreateInvite(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
}

func TestInviteHandler_CreateInvite_Returns401WithNoAuth(t *testing.T) {
	handler := org.NewInviteHandler(&mockInviteSvc{}, zerolog.Nop())
	o := &org.Org{ID: uuid.New(), Slug: "test-org", Plan: "teams"}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/invites",
		bytes.NewReader([]byte(`{"email":"x@x.com","role":"editor"}`)))
	req = withOrgContext(req, o)
	rr := httptest.NewRecorder()

	handler.CreateInvite(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestInviteHandler_AcceptInvite_Returns204(t *testing.T) {
	handler := org.NewInviteHandler(&mockInviteSvc{}, zerolog.Nop())

	req := makeSignedRequest(t, http.MethodPost, "/api/v1/invites/valid-token/accept", nil, "viewer")
	req.SetPathValue("token", "valid-token")
	rr := httptest.NewRecorder()

	handler.AcceptInvite(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rr.Code, rr.Body)
	}
}

func TestInviteHandler_AcceptInvite_Returns404ForBadToken(t *testing.T) {
	handler := org.NewInviteHandler(&mockInviteSvc{}, zerolog.Nop())

	req := makeSignedRequest(t, http.MethodPost, "/api/v1/invites/bad-token/accept", nil, "viewer")
	req.SetPathValue("token", "bad-token")
	rr := httptest.NewRecorder()

	handler.AcceptInvite(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body)
	}
}
