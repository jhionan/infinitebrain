// Package audit provides operational audit recording for RBAC actions and
// resource mutations.
package audit

import "context"

// Repository is the data access contract for audit_log.
// Insert is the only operation — audit_log is append-only.
type Repository interface {
	Insert(ctx context.Context, e *Entry) error
}
