// Package audit provides operational audit recording for RBAC actions and
// resource mutations. It is distinct from the compliance Logger in
// compliance.go, which is the SaaS tamper-evident log (T-104).
package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rian/infinite_brain/internal/auth"
)

// Recorder is the interface services and middleware use to record operational
// audit events (RBAC actions, resource mutations).
type Recorder interface {
	Record(ctx context.Context, action, targetType string, targetID *uuid.UUID, before, after any)
}

type auditRecorder struct {
	repo Repository
}

// NewRecorder returns a Recorder backed by the given repository.
func NewRecorder(repo Repository) Recorder {
	return &auditRecorder{repo: repo}
}

// Record appends an audit event. Silently skips if claims are not present in ctx.
// Fire-and-forget: the insert runs in a goroutine so audit failures never block
// the caller. Errors are intentionally dropped — the operation must proceed.
func (l *auditRecorder) Record(ctx context.Context, action, targetType string, targetID *uuid.UUID, before, after any) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return
	}
	e := buildEntry(claims, action, targetType, targetID, before, after)
	go l.repo.Insert(context.WithoutCancel(ctx), e) //nolint:errcheck // fire-and-forget by design
}

func buildEntry(claims *auth.Claims, action, targetType string, targetID *uuid.UUID, before, after any) *Entry {
	return &Entry{
		OrgID:      claims.OrgID,
		ActorID:    claims.UserID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     marshalJSON(before),
		After:      marshalJSON(after),
	}
}

// NoopRecorder discards all audit events. Used in tests and non-DB environments.
type NoopRecorder struct{}

// Record is a no-op.
func (NoopRecorder) Record(_ context.Context, _, _ string, _ *uuid.UUID, _, _ any) {}

// ContextWithClaims injects Claims into ctx using the same mechanism as Auth middleware.
// Exported for use in tests that need a claims-bearing context without a real HTTP round-trip.
func ContextWithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return auth.ContextWithClaims(ctx, claims)
}

func marshalJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}
