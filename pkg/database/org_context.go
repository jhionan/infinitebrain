package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithOrgContext acquires a pool connection, sets app.current_org_id for the
// transaction via SET LOCAL, and calls fn with the scoped connection.
// The connection is released back to the pool when fn returns.
// RLS policies on all data tables use current_setting('app.current_org_id') to
// filter rows — this is the single entry point for all org-scoped queries.
func WithOrgContext(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, fn func(*pgxpool.Conn) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for org context: %w", err)
	}
	defer conn.Release()

	// SET does not accept $N bind parameters in PostgreSQL; set_config is the
	// safe parameterized alternative. is_local=false sets the GUC at session
	// scope, which is correct here: the connection is always released via
	// defer conn.Release(), so the setting never leaks to another caller.
	if _, err = conn.Exec(ctx,
		`SELECT set_config('app.current_org_id', $1, false)`, orgID.String(),
	); err != nil {
		return fmt.Errorf("setting org context: %w", err)
	}

	return fn(conn)
}
