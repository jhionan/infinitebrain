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

// ── model tests ───────────────────────────────────────────────────────────────

func TestMarshalSettings_RoundTrip(t *testing.T) {
	s := org.OrgSettings{
		AIProvider:        "claude",
		RequireMFA:        true,
		DataRetentionDays: 90,
	}
	b, err := org.MarshalSettings(s)
	if err != nil {
		t.Fatalf("MarshalSettings: %v", err)
	}
	var got org.OrgSettings
	if err := org.UnmarshalSettings(b, &got); err != nil {
		t.Fatalf("UnmarshalSettings: %v", err)
	}
	if got.AIProvider != s.AIProvider || got.RequireMFA != s.RequireMFA || got.DataRetentionDays != s.DataRetentionDays {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, s)
	}
}

func TestUnmarshalSettings_EmptyInput_IsNoop(t *testing.T) {
	var s org.OrgSettings
	if err := org.UnmarshalSettings(nil, &s); err != nil {
		t.Errorf("unexpected error for nil input: %v", err)
	}
	if err := org.UnmarshalSettings([]byte{}, &s); err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}
}

// ── limits tests ──────────────────────────────────────────────────────────────

func TestLimitsFor_KnownPlan_ReturnsPlanLimits(t *testing.T) {
	l := org.LimitsFor("teams")
	if l.MaxMembers != 25 {
		t.Errorf("expected MaxMembers=25 for teams, got %d", l.MaxMembers)
	}
}

func TestLimitsFor_UnknownPlan_FallsBackToPersonal(t *testing.T) {
	l := org.LimitsFor("nonexistent-plan")
	personal := org.LimitsFor("personal")
	if l.MaxMembers != personal.MaxMembers {
		t.Errorf("expected fallback to personal limits, got MaxMembers=%d", l.MaxMembers)
	}
}

func TestCheckMemberLimit_UnlimitedPlan_ReturnsNil(t *testing.T) {
	if err := org.CheckMemberLimit("enterprise", 10000); err != nil {
		t.Errorf("expected nil for unlimited plan, got %v", err)
	}
}

func TestCheckMemberLimit_AtLimit_ReturnsError(t *testing.T) {
	if err := org.CheckMemberLimit("personal", 1); !errors.Is(err, apperrors.ErrPlanLimitReached) {
		t.Errorf("expected ErrPlanLimitReached at limit, got %v", err)
	}
}

func TestCheckMemberLimit_BelowLimit_ReturnsNil(t *testing.T) {
	if err := org.CheckMemberLimit("teams", 10); err != nil {
		t.Errorf("expected nil below limit, got %v", err)
	}
}

// ── mock repository ───────────────────────────────────────────────────────────

type mockRepo struct {
	orgs    map[string]*org.Org // keyed by slug
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
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation when removing last owner, got %v", err)
	}
}

func TestOrgService_UpdateMemberRole_CannotDemoteLastOwner(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "demote-org", Plan: "personal"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.UpdateMemberRole(context.Background(), o.ID, ownerID, ownerID, "admin")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation when demoting last owner, got %v", err)
	}
}

func TestOrgService_Update_SucceedsForAdmin(t *testing.T) {
	svc, repo := newTestService()
	adminID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "update-org", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, adminID, "admin", nil)

	updated, err := svc.Update(context.Background(), o.ID, adminID, "New Name", org.OrgSettings{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}
}

func TestOrgService_Update_RejectsForbiddenCaller(t *testing.T) {
	svc, repo := newTestService()
	viewerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "update-forbidden", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, viewerID, "viewer", nil)

	_, err := svc.Update(context.Background(), o.ID, viewerID, "New Name", org.OrgSettings{})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestOrgService_Update_RejectsEmptyName(t *testing.T) {
	svc, repo := newTestService()
	adminID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "update-emptyname", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, adminID, "admin", nil)

	_, err := svc.Update(context.Background(), o.ID, adminID, "", org.OrgSettings{})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation for empty name, got %v", err)
	}
}

func TestOrgService_Delete_SucceedsForOwner(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "delete-org", Plan: "personal"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	if err := svc.Delete(context.Background(), o.ID, ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrgService_Delete_RejectsForbiddenCaller(t *testing.T) {
	svc, repo := newTestService()
	adminID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "delete-forbidden", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, adminID, "admin", nil)

	err := svc.Delete(context.Background(), o.ID, adminID)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestOrgService_ListMembers_ReturnsMembers(t *testing.T) {
	svc, repo := newTestService()
	o := &org.Org{ID: uuid.New(), Slug: "list-org", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, uuid.New(), "editor", nil)
	_ = repo.AddMember(context.Background(), o.ID, uuid.New(), "viewer", nil)

	members, err := svc.ListMembers(context.Background(), o.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestOrgService_AddMember_RejectsInvalidRole(t *testing.T) {
	svc, repo := newTestService()
	o := &org.Org{ID: uuid.New(), Slug: "invalid-role", Plan: "teams"}
	repo.seedOrg(o)

	err := svc.AddMember(context.Background(), o.ID, uuid.New(), "superuser", uuid.New())
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation for invalid role, got %v", err)
	}
}

func TestOrgService_AddMember_RespectsMaxMembersOverride(t *testing.T) {
	svc, repo := newTestService()
	maxTwo := 2
	o := &org.Org{ID: uuid.New(), Slug: "max-override", Plan: "enterprise", MaxMembers: &maxTwo}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, uuid.New(), "editor", nil)
	_ = repo.AddMember(context.Background(), o.ID, uuid.New(), "editor", nil)

	err := svc.AddMember(context.Background(), o.ID, uuid.New(), "editor", uuid.New())
	if !errors.Is(err, apperrors.ErrPlanLimitReached) {
		t.Errorf("expected ErrPlanLimitReached for MaxMembers override, got %v", err)
	}
}

func TestOrgService_UpdateMemberRole_SucceedsWithTwoOwners(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	owner2ID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "two-owners", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)
	_ = repo.AddMember(context.Background(), o.ID, owner2ID, "owner", nil)

	// Demoting owner2 should be allowed since ownerID remains as owner.
	err := svc.UpdateMemberRole(context.Background(), o.ID, owner2ID, ownerID, "admin")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrgService_RemoveMember_SucceedsForNonLastOwner(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	editorID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "remove-ok", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)
	_ = repo.AddMember(context.Background(), o.ID, editorID, "editor", nil)

	if err := svc.RemoveMember(context.Background(), o.ID, editorID, ownerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrgService_RemoveMember_MemberRemovesThemselves(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	editorID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "self-remove", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)
	_ = repo.AddMember(context.Background(), o.ID, editorID, "editor", nil)

	// Editor removes themselves — no admin role check needed.
	if err := svc.RemoveMember(context.Background(), o.ID, editorID, editorID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrgService_RequireRole_RejectsMissingMember(t *testing.T) {
	svc, repo := newTestService()
	o := &org.Org{ID: uuid.New(), Slug: "require-role-org", Plan: "teams"}
	repo.seedOrg(o)

	// callerID is not a member at all — should get ErrForbidden.
	_, err := svc.Update(context.Background(), o.ID, uuid.New(), "x", org.OrgSettings{})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden for non-member caller, got %v", err)
	}
}

func TestOrgService_UpdateMemberRole_RejectsInvalidRole(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	targetID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "updaterole-invalid", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)
	_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

	err := svc.UpdateMemberRole(context.Background(), o.ID, targetID, ownerID, "superuser")
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation for invalid role, got %v", err)
	}
}

func TestOrgService_UpdateMemberRole_RejectsForbiddenCaller(t *testing.T) {
	svc, repo := newTestService()
	viewerID := uuid.New()
	targetID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "updaterole-forbidden", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, viewerID, "viewer", nil)
	_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

	err := svc.UpdateMemberRole(context.Background(), o.ID, targetID, viewerID, "viewer")
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestOrgService_UpdateMemberRole_RejectsMissingTarget(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "updaterole-notarget", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.UpdateMemberRole(context.Background(), o.ID, uuid.New(), ownerID, "editor")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing target, got %v", err)
	}
}

func TestOrgService_RemoveMember_RejectsForbiddenNonSelf(t *testing.T) {
	svc, repo := newTestService()
	viewerID := uuid.New()
	targetID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "remove-forbidden", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, viewerID, "viewer", nil)
	_ = repo.AddMember(context.Background(), o.ID, targetID, "editor", nil)

	// viewer trying to remove someone else — should be ErrForbidden.
	err := svc.RemoveMember(context.Background(), o.ID, targetID, viewerID)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestOrgService_AddMember_FailsWhenOrgNotFound(t *testing.T) {
	svc, _ := newTestService()

	err := svc.AddMember(context.Background(), uuid.New(), uuid.New(), "editor", uuid.New())
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing org, got %v", err)
	}
}

func TestOrgService_RemoveMember_FailsWhenTargetNotFound(t *testing.T) {
	svc, repo := newTestService()
	ownerID := uuid.New()
	o := &org.Org{ID: uuid.New(), Slug: "remove-no-target", Plan: "teams"}
	repo.seedOrg(o)
	_ = repo.AddMember(context.Background(), o.ID, ownerID, "owner", nil)

	err := svc.RemoveMember(context.Background(), o.ID, uuid.New(), ownerID)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing target, got %v", err)
	}
}
