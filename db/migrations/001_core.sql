-- Migration 001 — Core identity tables.
-- Generated from db/schema/001_core.hcl
-- Apply with: atlas schema apply --env local
--             or directly: psql $DATABASE_URL -f db/migrations/001_core.sql

BEGIN;

-- Extensions -------------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

-- orgs -------------------------------------------------------------------------

CREATE TABLE orgs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    slug          TEXT        NOT NULL UNIQUE,
    plan          TEXT        NOT NULL DEFAULT 'personal'
                              CHECK (plan IN ('personal', 'team', 'org', 'enterprise')),
    phi_enabled   BOOLEAN     NOT NULL DEFAULT false,
    baa_signed_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- users ------------------------------------------------------------------------

CREATE TABLE users (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    email          TEXT        NOT NULL UNIQUE,
    display_name   TEXT        NOT NULL,
    role           TEXT        NOT NULL DEFAULT 'member'
                               CHECK (role IN ('owner', 'admin', 'editor', 'viewer', 'member')),
    password_hash  TEXT,
    pepper_version SMALLINT    NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    last_active_at TIMESTAMPTZ
);

CREATE INDEX users_org_id_idx ON users (org_id);
CREATE INDEX users_email_idx  ON users (email);

-- org_units --------------------------------------------------------------------
-- Self-referencing tree. unit_type is free-form — no enum constraint.
-- The hierarchy is defined entirely by parent_unit_id.

CREATE TABLE org_units (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    parent_unit_id UUID        REFERENCES org_units(id) ON DELETE CASCADE,
    name           TEXT        NOT NULL,
    unit_type      TEXT        NOT NULL DEFAULT 'unit',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX org_units_org_id_idx        ON org_units (org_id);
CREATE INDEX org_units_parent_unit_id_idx ON org_units (parent_unit_id);
CREATE UNIQUE INDEX org_units_org_name_parent_key
    ON org_units (org_id, name, parent_unit_id);

COMMIT;
