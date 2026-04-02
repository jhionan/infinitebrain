// db/schema/006_rbac.hcl
// Migration 006 — RBAC: audit_log + org_invites tables.

// ── audit_log ─────────────────────────────────────────────────────────────────
// Append-only audit trail. Never UPDATE or DELETE rows from this table.
// Indexed on (org_id, created_at DESC) for paginated audit log queries.

table "audit_log" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "org_id" {
    null = false
    type = uuid
  }
  column "actor_id" {
    null = false
    type = uuid
  }
  column "action" {
    null = false
    type = text
  }
  column "target_type" {
    null = true
    type = text
  }
  column "target_id" {
    null = true
    type = uuid
  }
  column "before" {
    null = true
    type = jsonb
  }
  column "after" {
    null = true
    type = jsonb
  }
  column "ip" {
    null = true
    type = sql("inet")
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "audit_log_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "audit_log_actor_id_fkey" {
    columns     = [column.actor_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  index "audit_log_org_id_created_at_idx" {
    columns = [column.org_id, column.created_at]
  }
  index "audit_log_actor_id_idx" {
    columns = [column.actor_id]
  }
}

// ── org_invites ───────────────────────────────────────────────────────────────
// Pending invitations to join an org. Token is single-use and expires after 7 days.

table "org_invites" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "org_id" {
    null = false
    type = uuid
  }
  column "email" {
    null = false
    type = text
  }
  column "role" {
    null    = false
    type    = text
    default = "editor"
  }
  column "invited_by" {
    null = false
    type = uuid
  }
  column "token" {
    null = false
    type = text
  }
  column "expires_at" {
    null    = false
    type    = timestamptz
    default = sql("(now() + interval '7 days')")
  }
  column "accepted_at" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "org_invites_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "org_invites_invited_by_fkey" {
    columns     = [column.invited_by]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  index "org_invites_token_key" {
    columns = [column.token]
    unique  = true
  }
  index "org_invites_org_id_idx" {
    columns = [column.org_id]
  }
  check "org_invites_role_check" {
    expr = "role IN ('owner', 'admin', 'editor', 'viewer', 'member')"
  }
}
