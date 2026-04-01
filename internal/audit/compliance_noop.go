package audit

import "context"

// NoopLogger satisfies Logger without persisting any events.
// Used in OSS builds where the tamper-evident audit log is not configured.
type NoopLogger struct{}

// Log is a no-op.
func (NoopLogger) Log(_ context.Context, _ Event) {}
