package database_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/rian/infinite_brain/pkg/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// newTestDB starts a PostgreSQL container, applies migrations, and returns the pool.
// The container is terminated when the test finishes.
func newTestDB(t *testing.T) *pgxpool.Pool {
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

	// Phase 1: apply migrations using a plain pool (no pgvector AfterConnect hook).
	// The vector extension must exist before pgvector.RegisterTypes can look up its OID.
	bootstrapPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create bootstrap pool: %v", err)
	}
	if err := database.ApplyMigrations(ctx, bootstrapPool); err != nil {
		bootstrapPool.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	bootstrapPool.Close()

	// Phase 2: create the production pool with pgvector type registration.
	pool, err := database.New(ctx, database.DefaultConfig(dsn))
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestNew_ConnectsSuccessfully(t *testing.T) {
	pool := newTestDB(t)

	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("ping after setup: %v", err)
	}
}

func TestNew_RegistersPgvectorTypes(t *testing.T) {
	pool := newTestDB(t)
	ctx := context.Background()

	// Create an org and user as required by the nodes foreign keys.
	orgID := mustExec(t, ctx, pool,
		`INSERT INTO orgs (name, slug) VALUES ('test-org', 'test-org') RETURNING id`)
	unitID := mustExec(t, ctx, pool,
		`INSERT INTO org_units (org_id, name, unit_type)
		 VALUES ($1, 'root', 'org') RETURNING id`, orgID)
	userID := mustExec(t, ctx, pool,
		`INSERT INTO users (org_id, email, display_name)
		 VALUES ($1, 'test@example.com', 'Test User') RETURNING id`, orgID)

	// Insert a node with a VECTOR(1536) embedding.
	// If pgvector types were not registered, this would fail with a type mismatch.
	raw := make([]float32, 1536)
	for i := range raw {
		raw[i] = float32(i) / 1536.0
	}
	embedding := pgvector.NewVector(raw)

	var nodeID string
	err := pool.QueryRow(ctx,
		`INSERT INTO nodes (org_id, user_id, unit_id, type, title, embedding)
		 VALUES ($1, $2, $3, 'note', 'test node', $4) RETURNING id`,
		orgID, userID, unitID, embedding,
	).Scan(&nodeID)
	if err != nil {
		t.Fatalf("insert node with embedding: %v", err)
	}

	// Read the embedding back and verify it round-trips correctly.
	var retrieved pgvector.Vector
	err = pool.QueryRow(ctx,
		`SELECT embedding FROM nodes WHERE id = $1`, nodeID,
	).Scan(&retrieved)
	if err != nil {
		t.Fatalf("read embedding: %v", err)
	}
	if len(retrieved.Slice()) != 1536 {
		t.Errorf("expected 1536-dim embedding, got %d", len(retrieved.Slice()))
	}
}

func TestSchema_RLSBlocksCrossOrgAccess(t *testing.T) {
	pool := newTestDB(t)
	ctx := context.Background()

	// Create two orgs.
	orgA := mustExec(t, ctx, pool,
		`INSERT INTO orgs (name, slug) VALUES ('org-a', 'org-a') RETURNING id`)
	orgB := mustExec(t, ctx, pool,
		`INSERT INTO orgs (name, slug) VALUES ('org-b', 'org-b') RETURNING id`)

	// Create a unit and user for org A.
	unitA := mustExec(t, ctx, pool,
		`INSERT INTO org_units (org_id, name) VALUES ($1, 'root') RETURNING id`, orgA)
	userA := mustExec(t, ctx, pool,
		`INSERT INTO users (org_id, email, display_name)
		 VALUES ($1, 'a@example.com', 'User A') RETURNING id`, orgA)

	// Insert a node belonging to org A.
	nodeA := mustExec(t, ctx, pool,
		`INSERT INTO nodes (org_id, user_id, unit_id, type, title)
		 VALUES ($1, $2, $3, 'note', 'org-a secret') RETURNING id`,
		orgA, userA, unitA)

	// testcontainers creates 'infinitebrain' as a superuser, and superusers bypass
	// RLS even with FORCE ROW LEVEL SECURITY. Create a non-privileged role to
	// simulate the application user that IS subject to RLS policies.
	if _, err := pool.Exec(ctx, `CREATE ROLE rls_tester NOLOGIN`); err != nil {
		t.Fatalf("create rls_tester role: %v", err)
	}
	if _, err := pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO rls_tester`); err != nil {
		t.Fatalf("grant usage: %v", err)
	}
	if _, err := pool.Exec(ctx, `GRANT SELECT ON nodes TO rls_tester`); err != nil {
		t.Fatalf("grant select: %v", err)
	}

	// Acquire a single connection, switch to the non-superuser role, and set the
	// org context to org B. Org A's node must not be visible from org B's context.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SET ROLE rls_tester`); err != nil {
		t.Fatalf("set role: %v", err)
	}
	// set_config is used because SET does not accept $N bind parameters.
	if _, err := conn.Exec(ctx,
		`SELECT set_config('app.current_org_id', $1, false)`, orgB,
	); err != nil {
		t.Fatalf("set org context: %v", err)
	}

	var count int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM nodes WHERE id = $1`, nodeA,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("RLS failed: org B can see %d node(s) belonging to org A", count)
	}
}

func TestSchema_NodesSearchVectorGeneratedAutomatically(t *testing.T) {
	pool := newTestDB(t)
	ctx := context.Background()

	orgID := mustExec(t, ctx, pool,
		`INSERT INTO orgs (name, slug) VALUES ('sv-org', 'sv-org') RETURNING id`)
	unitID := mustExec(t, ctx, pool,
		`INSERT INTO org_units (org_id, name) VALUES ($1, 'root') RETURNING id`, orgID)
	userID := mustExec(t, ctx, pool,
		`INSERT INTO users (org_id, email, display_name)
		 VALUES ($1, 'sv@example.com', 'SV User') RETURNING id`, orgID)

	nodeID := mustExec(t, ctx, pool,
		`INSERT INTO nodes (org_id, user_id, unit_id, type, title, content)
		 VALUES ($1, $2, $3, 'note', 'Atlas migrations', 'schema as code with pgvector')
		 RETURNING id`,
		orgID, userID, unitID)

	// Verify the GENERATED ALWAYS column was populated automatically.
	var count int
	err := pool.QueryRow(ctx,
		`SELECT count(*) FROM nodes
		 WHERE id = $1
		 AND to_tsvector('english', title || ' ' || content) @@ to_tsquery('pgvector')`,
		nodeID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("full-text query: %v", err)
	}
	if count != 1 {
		t.Error("expected full-text search to match 'pgvector' in node content")
	}
}

func TestNew_InvalidDSN_ReturnsError(t *testing.T) {
	cfg := database.Config{DSN: "postgres://invalid:5432/nodb?sslmode=disable"}
	_, err := database.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

// mustExec runs a query that returns a single UUID and fails the test on error.
//
//nolint:revive // t *testing.T must be first per Go testing conventions
func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		t.Fatalf("mustExec(%q): %v", query, err)
	}
	return id
}
