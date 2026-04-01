package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rian/infinite_brain/pkg/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// mustTestPool starts a minimal PostgreSQL container and applies only the first
// migration. WithOrgContext does not require any application tables — it only
// sets a GUC — so 001_core.sql (which creates the pg extensions) is sufficient.
func mustTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	ctr, err := postgres.Run(ctx,
		"pgvector/pgvector:pg18",
		postgres.WithDatabase("infinitebrain_test"),
		postgres.WithUsername("infinitebrain"),
		postgres.WithPassword("infinitebrain"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	// Use a plain pool for bootstrapping — pgvector registration is unnecessary
	// for a GUC test but we apply 001_core.sql to ensure a valid schema state.
	bootstrapPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create bootstrap pool: %v", err)
	}
	if err := database.ApplyMigration(ctx, bootstrapPool, "001_core.sql"); err != nil {
		bootstrapPool.Close()
		t.Fatalf("apply 001_core.sql: %v", err)
	}
	bootstrapPool.Close()

	pool, err := database.New(ctx, database.DefaultConfig(dsn))
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestWithOrgContext_SetsOrgID(t *testing.T) {
	pool := mustTestPool(t)
	ctx := context.Background()

	orgID := uuid.New()

	err := database.WithOrgContext(ctx, pool, orgID, func(conn *pgxpool.Conn) error {
		var got string
		if err := conn.QueryRow(ctx,
			`SELECT current_setting('app.current_org_id', true)`,
		).Scan(&got); err != nil {
			return err
		}
		if got != orgID.String() {
			t.Errorf("current_setting: got %q, want %q", got, orgID.String())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithOrgContext: %v", err)
	}
}

func TestWithOrgContext_ReleasesConnectionOnSuccess(t *testing.T) {
	pool := mustTestPool(t)
	ctx := context.Background()

	before := pool.Stat().AcquiredConns()

	err := database.WithOrgContext(ctx, pool, uuid.New(), func(_ *pgxpool.Conn) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithOrgContext: %v", err)
	}

	after := pool.Stat().AcquiredConns()
	if after != before {
		t.Errorf("connection leak: acquired conns before=%d after=%d", before, after)
	}
}

func TestWithOrgContext_ReleasesConnectionOnError(t *testing.T) {
	pool := mustTestPool(t)
	ctx := context.Background()

	before := pool.Stat().AcquiredConns()

	_ = database.WithOrgContext(ctx, pool, uuid.New(), func(_ *pgxpool.Conn) error {
		return errSentinel
	})

	after := pool.Stat().AcquiredConns()
	if after != before {
		t.Errorf("connection leak on error: acquired conns before=%d after=%d", before, after)
	}
}

func TestWithOrgContext_DoesNotLeakAcrossPoolConnections(t *testing.T) {
	pool := mustTestPool(t)

	orgID := uuid.New()

	// Set org context in one call.
	err := database.WithOrgContext(context.Background(), pool, orgID, func(conn *pgxpool.Conn) error {
		return nil // just set it and release
	})
	if err != nil {
		t.Fatalf("first WithOrgContext failed: %v", err)
	}

	// Acquire a connection directly (may reuse the same pool connection).
	// The GUC must NOT carry over.
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire after WithOrgContext failed: %v", err)
	}
	defer conn.Release()

	var got string
	if err := conn.QueryRow(context.Background(),
		"SELECT current_setting('app.current_org_id', true)").Scan(&got); err != nil {
		t.Fatalf("query current_setting: %v", err)
	}

	// After the transaction committed, the transaction-local GUC is cleared.
	// The raw connection should have no org_id set (empty string with missing_ok=true).
	if got == orgID.String() {
		t.Errorf("GUC leaked across pool connection: expected empty, got %s", got)
	}
}

// errSentinel is a simple error value used to simulate fn returning an error.
var errSentinel = &sentinelError{}

type sentinelError struct{}

func (e *sentinelError) Error() string { return "sentinel error" }
