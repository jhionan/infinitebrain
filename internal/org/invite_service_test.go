package org_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/org"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// ── mock invite repo ──────────────────────────────────────────────────────────

type mockInviteRepo struct {
	invites map[string]*org.Invite // keyed by token
}

func newMockInviteRepo() *mockInviteRepo {
	return &mockInviteRepo{invites: make(map[string]*org.Invite)}
}

func (m *mockInviteRepo) Create(_ context.Context, i *org.Invite) (*org.Invite, error) {
	m.invites[i.Token] = i
	return i, nil
}

func (m *mockInviteRepo) FindByToken(_ context.Context, token string) (*org.Invite, error) {
	i, ok := m.invites[token]
	if !ok {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("invite not found"))
	}
	if i.AcceptedAt != nil {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("invite already accepted"))
	}
	if time.Now().After(i.ExpiresAt) {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("invite expired"))
	}
	return i, nil
}

func (m *mockInviteRepo) Accept(_ context.Context, id uuid.UUID) error {
	for _, i := range m.invites {
		if i.ID == id {
			now := time.Now()
			i.AcceptedAt = &now
			return nil
		}
	}
	return apperrors.ErrNotFound.Wrap(errors.New("invite not found"))
}

const testInviteRole = "editor"

// ── tests ─────────────────────────────────────────────────────────────────────

func newTestInviteService() (org.InviteService, *mockRepo, *mockInviteRepo) {
	orgRepo := newMockRepo()
	inviteRepo := newMockInviteRepo()
	svc := org.NewInviteService(inviteRepo, orgRepo)
	return svc, orgRepo, inviteRepo
}

func TestInviteService_CreateInvite_Succeeds(t *testing.T) {
	svc, orgRepo, _ := newTestInviteService()
	callerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "invite-org", Plan: "teams"}
	orgRepo.seedOrg(o)
	_ = orgRepo.AddMember(context.Background(), o.ID, callerID, "admin", nil)

	invite, err := svc.CreateInvite(context.Background(), o.ID, "new@example.com", testInviteRole, callerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got %q", invite.Email)
	}
	if invite.Role != testInviteRole {
		t.Errorf("expected role 'editor', got %q", invite.Role)
	}
	if invite.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestInviteService_CreateInvite_RejectsForbiddenCaller(t *testing.T) {
	svc, orgRepo, _ := newTestInviteService()
	viewerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "invite-forbidden", Plan: "teams"}
	orgRepo.seedOrg(o)
	_ = orgRepo.AddMember(context.Background(), o.ID, viewerID, "viewer", nil)

	_, err := svc.CreateInvite(context.Background(), o.ID, "new@example.com", testInviteRole, viewerID)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestInviteService_CreateInvite_RejectsInvalidRole(t *testing.T) {
	svc, orgRepo, _ := newTestInviteService()
	adminID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "invite-badrole", Plan: "teams"}
	orgRepo.seedOrg(o)
	_ = orgRepo.AddMember(context.Background(), o.ID, adminID, "admin", nil)

	_, err := svc.CreateInvite(context.Background(), o.ID, "new@example.com", "superuser", adminID)
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation for invalid role, got %v", err)
	}
}

func TestInviteService_AcceptInvite_AddsUserToOrg(t *testing.T) {
	svc, orgRepo, inviteRepo := newTestInviteService()
	callerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "accept-org", Plan: "teams"}
	orgRepo.seedOrg(o)
	_ = orgRepo.AddMember(context.Background(), o.ID, callerID, "admin", nil)

	token := "test-token-abc123"
	inviteRepo.invites[token] = &org.Invite{
		ID:        uuid.New(),
		OrgID:     o.ID,
		Email:     "newuser@example.com",
		Role:      testInviteRole,
		InvitedBy: callerID,
		Token:     token,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	acceptingUser := uuid.New()
	if err := svc.AcceptInvite(context.Background(), token, acceptingUser); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	member, err := orgRepo.FindMember(context.Background(), o.ID, acceptingUser)
	if err != nil {
		t.Fatalf("expected member to be added: %v", err)
	}
	if member.Role != testInviteRole {
		t.Errorf("expected role 'editor', got %q", member.Role)
	}
}

func TestInviteService_AcceptInvite_ReturnsNotFoundForInvalidToken(t *testing.T) {
	svc, _, _ := newTestInviteService()

	err := svc.AcceptInvite(context.Background(), "no-such-token", uuid.New())
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Compile check: auth exports are accessible from org_test.
var _ = auth.Can
var _ auth.Permission = auth.PermManageMembers
