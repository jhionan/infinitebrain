// Package audit defines the compliance audit log interface.
// The noop implementation is compiled into OSS builds.
// The real tamper-evident implementation lives in infinitebrain-cloud.
package audit

import "context"

// EventKind classifies audit log entries.
type EventKind string

const (
	// EventKindCreate is logged when a resource is created.
	EventKindCreate EventKind = "create"
	// EventKindUpdate is logged when a resource is updated.
	EventKindUpdate EventKind = "update"
	// EventKindDelete is logged when a resource is deleted.
	EventKindDelete EventKind = "delete"
	// EventKindRead is logged when PHI is accessed.
	EventKindRead EventKind = "read"
	// EventKindAuth is logged for authentication events.
	EventKindAuth EventKind = "auth"
)

// Event is a single audit log entry.
type Event struct {
	Kind       EventKind
	ActorID    string
	ResourceID string
	OrgID      string
	IsPHI      bool
	Metadata   map[string]string
}

// Logger appends immutable audit events.
// Implementations must be append-only — no update or delete operations.
type Logger interface {
	// Log appends an audit event. Implementations must never block the caller
	// and must never return an error that would abort a business operation.
	Log(ctx context.Context, event Event)
}
