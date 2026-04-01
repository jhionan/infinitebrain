// Migration 003 — Multi-tenancy: org membership + RLS.
// org_members maps users to orgs with a role.
// RLS is enabled on nodes so every query is auto-scoped to current org.

// ── org_members ───────────────────────────────────────────────────────────────
// Maps users to orgs. A user can belong to many orgs.
// The role here is the org-level role (owner/admin/editor/viewer).
// RBAC rules (T-102) build on top of this.

table "org_members" {
  schema = schema.public

  column "org_id" {
    null = false
    type = uuid
  }
  column "user_id" {
    null = false
    type = uuid
  }
  column "role" {
    null    = false
    type    = text
    default = "member"
  }
  column "invited_by" {
    null = true
    type = uuid
  }
  column "joined_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.org_id, column.user_id]
  }
  foreign_key "org_members_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "org_members_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "org_members_invited_by_fkey" {
    columns     = [column.invited_by]
    ref_columns = [table.users.column.id]
    on_delete   = SET_NULL
  }
  index "org_members_user_id_idx" {
    columns = [column.user_id]
  }
  check "org_members_role_check" {
    expr = "role IN ('owner', 'admin', 'editor', 'viewer', 'member')"
  }
}

// ── Row-Level Security ────────────────────────────────────────────────────────
// Enable RLS on nodes and create the org isolation policy.
// The application sets app.current_org_id per connection via WithOrgContext.
// Using a non-superuser app role ensures RLS is never bypassed.
//
// NOTE: Atlas HCL does not natively support CREATE POLICY syntax.
// The RLS enablement and policy creation lives in the SQL migration file.
