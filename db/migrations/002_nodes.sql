-- Migration 002 — Universal knowledge graph (nodes + edges).
-- Generated from db/schema/002_nodes.hcl
-- Depends on: 001_core.sql

BEGIN;

-- nodes ------------------------------------------------------------------------

CREATE TABLE nodes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unit_id         UUID        NOT NULL REFERENCES org_units(id) ON DELETE CASCADE,

    -- Content
    type            TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    content         TEXT,
    content_enc     BYTEA,

    -- Classification
    para            TEXT        CHECK (para IN ('project', 'area', 'resource', 'archive')),
    project_id      UUID        REFERENCES nodes(id) ON DELETE SET NULL,
    tags            TEXT[]      NOT NULL DEFAULT '{}',

    -- Privacy — default is 'individual', system never auto-escalates
    visibility      TEXT        NOT NULL DEFAULT 'individual'
                                CHECK (visibility IN ('individual', 'unit', 'unit_and_above', 'org', 'public')),
    is_phi          BOOLEAN     NOT NULL DEFAULT false,

    -- Intelligence
    embedding       VECTOR(1536),
    search_vector   TSVECTOR    GENERATED ALWAYS AS (
                        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
                    ) STORED,

    -- Spaced repetition (T-036)
    review_stage    SMALLINT    NOT NULL DEFAULT 0,
    next_review_at  TIMESTAMPTZ,

    -- Deduplication (T-128)
    dedup_dismissed BOOLEAN     NOT NULL DEFAULT false,

    -- Standard
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    archived_at     TIMESTAMPTZ
);

-- Standard query indexes
CREATE INDEX nodes_org_user_idx        ON nodes (org_id, user_id);
CREATE INDEX nodes_org_type_idx        ON nodes (org_id, type);
CREATE INDEX nodes_unit_id_idx         ON nodes (unit_id);
CREATE INDEX nodes_org_visibility_idx  ON nodes (org_id, visibility);
CREATE INDEX nodes_org_user_para_idx   ON nodes (org_id, user_id, para);
CREATE INDEX nodes_project_id_idx      ON nodes (project_id);
CREATE INDEX nodes_org_user_review_idx ON nodes (org_id, user_id, review_stage, next_review_at);

-- Full-text search
CREATE INDEX nodes_search_vector_gin_idx ON nodes USING gin (search_vector);

-- Tag filtering
CREATE INDEX nodes_tags_gin_idx ON nodes USING gin (tags);

-- HNSW approximate nearest-neighbour (pgvector).
-- m=16 and ef_construction=64 are balanced defaults (recall vs speed).
CREATE INDEX nodes_embedding_hnsw_idx ON nodes
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Row-Level Security -----------------------------------------------------------
-- Org isolation enforced at the database layer.
-- app.current_org_id is set per-connection by the auth middleware.

ALTER TABLE nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE nodes FORCE ROW LEVEL SECURITY;

CREATE POLICY nodes_org_isolation ON nodes
    USING (
        current_setting('app.current_org_id', true) != ''
        AND org_id = current_setting('app.current_org_id', true)::uuid
    );

-- edges ------------------------------------------------------------------------

CREATE TABLE edges (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    from_node_id  UUID        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node_id    UUID        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    relation_type TEXT        NOT NULL,
    confidence    FLOAT       NOT NULL DEFAULT 1.0
                              CHECK (confidence BETWEEN 0.0 AND 1.0),
    created_by    TEXT        NOT NULL
                              CHECK (created_by IN ('user', 'ai', 'insight-linker')),
    metadata      JSONB       NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (from_node_id, to_node_id, relation_type)
);

CREATE INDEX edges_org_id_idx       ON edges (org_id);
CREATE INDEX edges_from_node_id_idx ON edges (from_node_id);
CREATE INDEX edges_to_node_id_idx   ON edges (to_node_id);
CREATE INDEX edges_org_relation_idx ON edges (org_id, relation_type);

ALTER TABLE edges ENABLE ROW LEVEL SECURITY;
ALTER TABLE edges FORCE ROW LEVEL SECURITY;

CREATE POLICY edges_org_isolation ON edges
    USING (
        current_setting('app.current_org_id', true) != ''
        AND org_id = current_setting('app.current_org_id', true)::uuid
    );

COMMIT;
