// internal/audit/logger_test.go
package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/audit"
	"github.com/rian/infinite_brain/internal/auth"
)

type mockAuditRepo struct {
	entries []*audit.Entry
}

func (m *mockAuditRepo) Insert(_ context.Context, e *audit.Entry) error {
	m.entries = append(m.entries, e)
	return nil
}

func claimsCtx(orgID, userID uuid.UUID) context.Context {
	claims := &auth.Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   "admin",
	}
	return audit.ContextWithClaims(context.Background(), claims)
}

func TestAuditRecorder_Record_CallsRepoInsert(t *testing.T) {
	repo := &mockAuditRepo{}
	recorder := audit.NewRecorder(repo)
	orgID := uuid.New()
	userID := uuid.New()
	ctx := claimsCtx(orgID, userID)

	targetID := uuid.New()
	recorder.Record(ctx, "node.delete", "node", &targetID, nil, nil)

	// Fire-and-forget uses a goroutine — sleep briefly to let it run.
	time.Sleep(10 * time.Millisecond)

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(repo.entries))
	}
	e := repo.entries[0]
	if e.OrgID != orgID {
		t.Errorf("expected org_id %s, got %s", orgID, e.OrgID)
	}
	if e.ActorID != userID {
		t.Errorf("expected actor_id %s, got %s", userID, e.ActorID)
	}
	if e.Action != "node.delete" {
		t.Errorf("expected action 'node.delete', got %q", e.Action)
	}
	if e.TargetType != "node" {
		t.Errorf("expected target_type 'node', got %q", e.TargetType)
	}
	if e.TargetID == nil || *e.TargetID != targetID {
		t.Errorf("expected target_id %s, got %v", targetID, e.TargetID)
	}
}

func TestAuditRecorder_Record_NoopWhenNoClaims(t *testing.T) {
	repo := &mockAuditRepo{}
	recorder := audit.NewRecorder(repo)

	recorder.Record(context.Background(), "node.delete", "node", nil, nil, nil)
	time.Sleep(10 * time.Millisecond)

	if len(repo.entries) != 0 {
		t.Errorf("expected 0 entries with no claims, got %d", len(repo.entries))
	}
}

func TestNoopRecorder_Record_DoesNotPanic(_ *testing.T) {
	var recorder audit.Recorder = audit.NoopRecorder{}
	recorder.Record(context.Background(), "any.action", "node", nil, nil, nil)
}

func TestAuditRecorder_Record_WithBeforeAndAfter(t *testing.T) {
	repo := &mockAuditRepo{}
	recorder := audit.NewRecorder(repo)
	orgID := uuid.New()
	userID := uuid.New()
	ctx := claimsCtx(orgID, userID)

	type payload struct {
		Name string `json:"name"`
	}
	recorder.Record(ctx, "node.update", "node", nil,
		payload{Name: "old"}, payload{Name: "new"})

	time.Sleep(10 * time.Millisecond)

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(repo.entries))
	}
	e := repo.entries[0]
	if e.Before == nil {
		t.Error("expected Before to be non-nil")
	}
	if e.After == nil {
		t.Error("expected After to be non-nil")
	}
}
