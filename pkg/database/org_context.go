package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithOrgContext acquires a pool connection, opens a transaction, sets
// app.current_org_id as a transaction-local GUC, and calls fn with the
// scoped connection. The GUC is cleared automatically when the transaction
// ends (COMMIT or ROLLBACK), so the connection returns to the pool in a
// clean state with no org context leak.
//
// RLS policies on data tables use current_setting('app.current_org_id') to
// filter rows — this is the single entry point for all org-scoped queries.
// The application role must NOT be a superuser (superusers bypass RLS).
func WithOrgContext(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, fn func(*pgxpool.Conn) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for org context: %w", err)
	}
	defer conn.Release()

	if _, err = conn.Exec(ctx, "BEGIN"); err != nil {
		return fmt.Errorf("begin org context transaction: %w", err)
	}

	if _, err = conn.Exec(ctx, `SELECT set_config('app.current_org_id', $1, true)`, orgID.String()); err != nil {
		conn.Exec(ctx, "ROLLBACK") //nolint:errcheck
		return fmt.Errorf("setting org context: %w", err)
	}

	fnErr := fn(conn)

	if _, err = conn.Exec(ctx, "COMMIT"); err != nil {
		conn.Exec(ctx, "ROLLBACK") //nolint:errcheck
		if fnErr != nil {
			return fnErr
		}
		return fmt.Errorf("commit org context transaction: %w", err)
	}

	return fnErr
}
