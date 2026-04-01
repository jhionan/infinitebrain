package audit_test

import (
	"context"
	"testing"

	"github.com/rian/infinite_brain/internal/audit"
)

func TestNoopLogger_Log_DoesNotPanic(t *testing.T) {
	var logger audit.Logger = audit.NoopLogger{}
	// Must not panic for any event kind.
	for _, kind := range []audit.EventKind{
		audit.EventKindCreate,
		audit.EventKindUpdate,
		audit.EventKindDelete,
		audit.EventKindRead,
		audit.EventKindAuth,
	} {
		logger.Log(context.Background(), audit.Event{
			Kind:       kind,
			ActorID:    "user-1",
			ResourceID: "res-1",
			OrgID:      "org-1",
		})
	}
}

func TestNoopLogger_ImplementsLogger(t *testing.T) {
	// Compile-time check that NoopLogger satisfies the Logger interface.
	var _ audit.Logger = audit.NoopLogger{}
}
