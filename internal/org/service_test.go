package org_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/org"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// ── mock repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	orgs    map[string]*org.Org   // keyed by slug
	orgsID  map[uuid.UUID]*org.Org
	members map[uuid.UUID][]org.Member // keyed by orgID
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		orgs:    make(map[string]*org.Org),
		orgsID:  make(map[uuid.UUID]*org.Org),
		members: make(map[uuid.UUID][]org.Member),
	}
}

func (m *mockRepo) seedOrg(o *org.Org) {
	m.orgs[o.Slug] = o
	m.orgsID[o.ID] = o
}

func (m *mockRepo) FindByID(_ context.Context, id uuid.UUID) (*org.Org, error) {
	o, ok := m.orgsID[id]
	if !ok {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("org not found"))
	}
	return o, nil
}
func (m *mockRepo) FindBySlug(_ context.Context, slug string) (*org.Org, error) {
	o, ok := m.orgs[slug]
	if !ok {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("org not found"))
	}
	return o, nil
}
func (m *mockRepo) Update(_ context.Context, id uuid.UUID, name string, settings org.OrgSettings) (*org.Org, error) {
	o, ok := m.orgsID[id]
	if !ok {
		return nil, apperrors.ErrNotFound.Wrap(errors.New("org not found"))
	}
	o.Name = name
	o.Settings = settings
	return o, nil
}
func (m *mockRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.orgsID, id)
	return nil
}
func (m *mockRepo) AddMember(_ context.Context, orgID, userID uuid.UUID, role string, _ *uuid.UUID) error {
	m.members[orgID] = append(m.members[orgID], org.Member{OrgID: orgID, UserID: userID, Role: role, JoinedAt: time.Now()})
	return nil
}
func (m *mockRepo) FindMember(_ context.Context, orgID, userID uuid.UUID) (*org.Member, error) {
	for _, mem := range m.members[orgID] {
		if mem.UserID == userID {
			return &mem, nil
		}
	}
	return nil, apperrors.ErrNotFound.Wrap(errors.New("member not found"))
}
func (m *mockRepo) ListMembers(_ context.Context, orgID uuid.UUID) ([]org.Member, error) {
	return m.members[orgID], nil
}
func (m *mockRepo) UpdateMemberRole(_ context.Context, orgID, userID uuid.UUID, role string) error {
	for i, mem := range m.members[orgID] {
		if mem.UserID == userID {
			m.members[orgID][i].Role = role
			return nil
		}
	}
	return apperrors.ErrNotFound.Wrap(errors.New("member not found"))
}
func (m *mockRepo) RemoveMember(_ context.Context, orgID, userID uuid.UUID) error {
	updated := m.members[orgID][:0]
	for _, mem := range m.members[orgID] {
		if mem.UserID != userID {
			updated = append(updated, mem)
		}
	}
	m.members[orgID] = updated
	return nil
}
func (m *mockRepo) CountMembers(_ context.Context, orgID uuid.UUID) (int64, error) {
	return int64(len(m.members[orgID])), nil
}

func newTestService() (org.Service, *mockRepo) {
	r := newMockRepo()
	return org.NewService(r), r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestOrgService_Get_ReturnsOrg(t *testing.T) {
	svc, repo := newTestService()
	o := &org.Org{ID: uuid.New(), Name: "Acme", Slug: "acme", Plan: "teams"}
	repo.seedOrg(o)

	got, err := svc.Get(context.Background(), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != o.ID {
		t.Errorf("expected id %s, got %s", o.ID, got.ID)
	}
}

func TestOrgService_Get_ReturnsNotFoundForMissing(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.Get(context.Background(), "no-such-org")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOrgService_AddMember_SucceedsUnderLimit(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "team1", Plan: "teams"}
	repo.seedOrg(o)
	// teams plan allows 25 members; adding first one should succeed
	if err := svc.AddMember(context.Background(), o.ID, uuid.New(), "editor", ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrgService_AddMember_FailsAtPersonalPlanLimit(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "personal1", Plan: "personal"}
	repo.seedOrg(o)
	// personal plan max = 1 member; seed the owner first
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.AddMember(context.Background(), o.ID, uuid.New(), "viewer", ownerID)
	if !errors.Is(err, apperrors.ErrPlanLimitReached) {
		t.Errorf("expected ErrPlanLimitReached, got %v", err)
	}
}

func TestOrgService_RemoveMember_CannotRemoveLastOwner(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "solo-org", Plan: "personal"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.RemoveMember(context.Background(), o.ID, ownerID, ownerID)
	if err == nil {
		t.Fatal("expected error when removing last owner, got nil")
	}
}

func TestOrgService_UpdateMemberRole_CannotDemoteLastOwner(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "demote-org", Plan: "personal"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.UpdateMemberRole(context.Background(), o.ID, ownerID, ownerID, "admin")
	if err == nil {
		t.Fatal("expected error when demoting last owner, got nil")
	}
}
