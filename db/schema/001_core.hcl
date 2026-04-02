// Migration 001 — Core identity tables.
// orgs → users → org_units
// Every other table references org_id from this schema.

schema "public" {}

// ── Extensions ────────────────────────────────────────────────────────────────

extension "pgcrypto" {
  schema  = schema.public
  comment = "gen_random_uuid() for primary keys"
}

extension "vector" {
  schema  = schema.public
  comment = "pgvector: HNSW similarity search for embeddings"
}

// ── orgs ──────────────────────────────────────────────────────────────────────
// Top-level tenant boundary. All data is scoped to an org.
// Every user gets a personal org on signup.

table "orgs" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "name" {
    null = false
    type = text
  }
  column "slug" {
    null = false
    type = text
  }
  column "plan" {
    null    = false
    type    = text
    default = "personal"
  }
  column "max_members" {
    null = true
    type = integer
  }
  column "settings" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
  }
  column "phi_enabled" {
    null    = false
    type    = boolean
    default = false
  }
  column "baa_signed_at" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }
  index "orgs_slug_key" {
    columns = [column.slug]
    unique  = true
  }
  check "orgs_plan_check" {
    expr = "plan IN ('personal', 'pro', 'teams', 'enterprise')"
  }
}

// ── users ─────────────────────────────────────────────────────────────────────
// One user belongs to one primary org (personal). Multi-org membership is
// handled via org_members (T-101). password_hash is NULL for OIDC-only users.

table "users" {
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
  column "display_name" {
    null = false
    type = text
  }
  column "role" {
    null    = false
    type    = text
    default = "member"
  }
  column "password_hash" {
    null = true
    type = text
  }
  // Tracks which pepper version was used to hash this password.
  // Increment when pepper is rotated; triggers re-hash on next login.
  column "pepper_version" {
    null    = false
    type    = smallint
    default = 1
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  column "last_active_at" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "users_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  index "users_org_id_idx" {
    columns = [column.org_id]
  }
  index "users_email_key" {
    columns = [column.email]
    unique  = true
  }
  check "users_role_check" {
    expr = "role IN ('owner', 'admin', 'editor', 'viewer', 'member')"
  }
}

// ── org_units ─────────────────────────────────────────────────────────────────
// Self-referencing tree. Represents the hierarchy of an org.
// unit_type is free-form — users choose their own names (team, squad, circle…).
// The hierarchy is determined by parent_unit_id, not by type values.
// Every org has a root unit (type = 'org'). Every user has an implicit
// personal unit (type = 'individual') as a child of the root.

table "org_units" {
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
  column "parent_unit_id" {
    null = true
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "unit_type" {
    null    = false
    type    = text
    default = "unit"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "org_units_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "org_units_parent_unit_id_fkey" {
    columns     = [column.parent_unit_id]
    ref_columns = [table.org_units.column.id]
    on_delete   = CASCADE
  }
  index "org_units_org_id_idx" {
    columns = [column.org_id]
  }
  index "org_units_parent_unit_id_idx" {
    columns = [column.parent_unit_id]
  }
  index "org_units_org_name_parent_key" {
    columns = [column.org_id, column.name, column.parent_unit_id]
    unique  = true
  }
}
