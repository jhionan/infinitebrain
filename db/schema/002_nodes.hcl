// Migration 002 — Universal knowledge graph.
// Every entity in Infinite Brain is a node. Every relationship is a typed edge.
// This is the primary data model — all features build on top of it.

// ── nodes ─────────────────────────────────────────────────────────────────────
// Universal content unit. Tasks, notes, decisions, specs, contacts — all nodes.
// The `type` column determines behavior; the schema is shared.

table "nodes" {
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
  column "user_id" {
    null = false
    type = uuid
  }
  column "unit_id" {
    null = false
    type = uuid
  }

  // ── Content ───────────────────────────────────────────────────────────────

  column "type" {
    null = false
    type = text
  }
  column "title" {
    null = false
    type = text
  }
  column "content" {
    null = true
    type = text
  }
  // AES-256-GCM ciphertext. Set when is_phi = true; content must be NULL.
  column "content_enc" {
    null = true
    type = bytea
  }

  // ── Classification ────────────────────────────────────────────────────────

  column "para" {
    null = true
    type = text
  }
  // Parent project node. NULL for top-level nodes.
  column "project_id" {
    null = true
    type = uuid
  }
  column "tags" {
    null    = false
    type    = sql("text[]")
    default = sql("'{}'::text[]")
  }

  // ── Hierarchy scope ───────────────────────────────────────────────────────

  // ── Privacy ───────────────────────────────────────────────────────────────
  // Default is 'individual'. System never auto-escalates.
  // Publishing to 'org' is irreversible (content is de-attributed at org layer).

  column "visibility" {
    null    = false
    type    = text
    default = "individual"
  }
  column "is_phi" {
    null    = false
    type    = boolean
    default = false
  }

  // ── Intelligence ──────────────────────────────────────────────────────────

  // 1536-dim vector from text-embedding-3-small. NULL until AI processing.
  column "embedding" {
    null = true
    type = sql("vector(1536)")
  }
  // GENERATED from title + content. Updated automatically by PostgreSQL.
  column "search_vector" {
    null = true
    type = tsvector
    as {
      expr = "to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))"
      type = STORED
    }
  }

  // ── Spaced repetition (T-036) ─────────────────────────────────────────────

  column "review_stage" {
    null    = false
    type    = smallint
    default = 0
  }
  column "next_review_at" {
    null = true
    type = timestamptz
  }

  // ── Deduplication (T-128) ────────────────────────────────────────────────

  column "dedup_dismissed" {
    null    = false
    type    = boolean
    default = false
  }

  // ── Standard ──────────────────────────────────────────────────────────────

  column "metadata" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
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
  column "archived_at" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "nodes_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "nodes_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "nodes_unit_id_fkey" {
    columns     = [column.unit_id]
    ref_columns = [table.org_units.column.id]
    on_delete   = CASCADE
  }
  foreign_key "nodes_project_id_fkey" {
    columns     = [column.project_id]
    ref_columns = [table.nodes.column.id]
    on_delete   = SET_NULL
  }

  // Composite index for the most common read pattern.
  index "nodes_org_user_idx" {
    columns = [column.org_id, column.user_id]
  }
  index "nodes_org_type_idx" {
    columns = [column.org_id, column.type]
  }
  index "nodes_unit_id_idx" {
    columns = [column.unit_id]
  }
  index "nodes_org_visibility_idx" {
    columns = [column.org_id, column.visibility]
  }
  index "nodes_org_user_para_idx" {
    columns = [column.org_id, column.user_id, column.para]
  }
  index "nodes_project_id_idx" {
    columns = [column.project_id]
  }
  index "nodes_org_user_review_idx" {
    columns = [column.org_id, column.user_id, column.review_stage, column.next_review_at]
  }
  // Full-text search (GIN).
  index "nodes_search_vector_gin_idx" {
    columns = [column.search_vector]
    type    = GIN
  }
  // Tag filtering (GIN).
  index "nodes_tags_gin_idx" {
    columns = [column.tags]
    type    = GIN
  }
  // HNSW approximate nearest-neighbour search for embeddings.
  // Parameters (m=16, ef_construction=64) are set post-migration via SQL.
  // See db/migrations/002_nodes_hnsw.sql
  index "nodes_embedding_hnsw_idx" {
    on {
      column = column.embedding
      ops    = sql("vector_cosine_ops")
    }
    type = HNSW
  }

  check "nodes_para_check" {
    expr = "para IS NULL OR para IN ('project', 'area', 'resource', 'archive')"
  }
  check "nodes_visibility_check" {
    expr = "visibility IN ('individual', 'unit', 'unit_and_above', 'org', 'public')"
  }
  // PHI constraint: when is_phi, content must be NULL and content_enc must be set.
  // Enforced in application layer (repository). Documented here as architecture intent.
}

// ── edges ─────────────────────────────────────────────────────────────────────
// Typed relationships between nodes.
// Visibility is not stored on edges — an edge is visible only if the requesting
// user has read access to BOTH from_node and to_node (enforced in service layer).

table "edges" {
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
  column "from_node_id" {
    null = false
    type = uuid
  }
  column "to_node_id" {
    null = false
    type = uuid
  }
  column "relation_type" {
    null = false
    type = text
  }
  // 1.0 = user-asserted. < 1.0 = AI-inferred with confidence score.
  column "confidence" {
    null    = false
    type    = float
    default = 1.0
  }
  column "created_by" {
    null = false
    type = text
  }
  column "metadata" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "edges_org_id_fkey" {
    columns     = [column.org_id]
    ref_columns = [table.orgs.column.id]
    on_delete   = CASCADE
  }
  foreign_key "edges_from_node_id_fkey" {
    columns     = [column.from_node_id]
    ref_columns = [table.nodes.column.id]
    on_delete   = CASCADE
  }
  foreign_key "edges_to_node_id_fkey" {
    columns     = [column.to_node_id]
    ref_columns = [table.nodes.column.id]
    on_delete   = CASCADE
  }
  index "edges_org_id_idx" {
    columns = [column.org_id]
  }
  index "edges_from_node_id_idx" {
    columns = [column.from_node_id]
  }
  index "edges_to_node_id_idx" {
    columns = [column.to_node_id]
  }
  index "edges_org_relation_idx" {
    columns = [column.org_id, column.relation_type]
  }
  // Prevents duplicate edges of the same type between two nodes.
  index "edges_from_to_relation_key" {
    columns = [column.from_node_id, column.to_node_id, column.relation_type]
    unique  = true
  }
  check "edges_confidence_check" {
    expr = "confidence BETWEEN 0.0 AND 1.0"
  }
  check "edges_created_by_check" {
    expr = "created_by IN ('user', 'ai', 'insight-linker')"
  }
}

// ── Row-Level Security ────────────────────────────────────────────────────────
// Enforced at the database level. A compromised application layer cannot leak
// another org's data even with unrestricted SQL access.
// The application sets app.current_org_id per connection via auth middleware.
//
// RLS policies are defined here for documentation. Atlas applies them as part
// of the schema diff. Policies use PERMISSIVE (default) mode — a row is
// accessible if it passes any policy.
