// Package audit provides operational audit recording for RBAC actions and
// resource mutations.
package audit

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	sqlcdb "github.com/rian/infinite_brain/db/sqlc"
)

type pgRepository struct {
	queries *sqlcdb.Queries
}

// NewRepository returns a PostgreSQL-backed audit Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{queries: sqlcdb.New(pool)}
}

func (r *pgRepository) Insert(ctx context.Context, e *Entry) error {
	var targetType *string
	if e.TargetType != "" {
		targetType = &e.TargetType
	}
	var targetID pgtype.UUID
	if e.TargetID != nil {
		targetID = pgtype.UUID{Bytes: *e.TargetID, Valid: true}
	}
	var ip *netip.Addr
	if e.IP != "" {
		parsed, err := netip.ParseAddr(e.IP)
		if err == nil {
			ip = &parsed
		}
	}

	err := r.queries.InsertAuditLog(ctx, sqlcdb.InsertAuditLogParams{
		OrgID:      pgtype.UUID{Bytes: e.OrgID, Valid: true},
		ActorID:    pgtype.UUID{Bytes: e.ActorID, Valid: true},
		Action:     e.Action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     e.Before,
		After:      e.After,
		Ip:         ip,
	})
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
