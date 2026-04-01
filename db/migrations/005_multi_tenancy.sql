-- Migration 005 — Multi-tenancy: org membership + RLS.
-- Adds max_members/settings to orgs, fixes plan CHECK, creates org_members table.
-- Generated from db/schema/005_multi_tenancy.hcl

BEGIN;

-- orgs: add max_members + settings columns, fix plan CHECK ----------------------

ALTER TABLE orgs
    ADD COLUMN max_members INTEGER,
    ADD COLUMN settings     JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE orgs DROP CONSTRAINT IF EXISTS orgs_plan_check;
ALTER TABLE orgs
    ADD CONSTRAINT orgs_plan_check
    CHECK (plan IN ('personal', 'pro', 'teams', 'enterprise'));

-- org_members ------------------------------------------------------------------
-- Maps users to orgs. A user can belong to many orgs.

CREATE TABLE org_members (
    org_id     UUID        NOT NULL REFERENCES orgs(id)  ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT        NOT NULL DEFAULT 'member'
                           CHECK (role IN ('owner', 'admin', 'editor', 'viewer', 'member')),
    invited_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX org_members_user_id_idx ON org_members (user_id);

-- Row-Level Security on nodes --------------------------------------------------
-- Enable RLS on nodes. The application sets app.current_org_id via WithOrgContext.
-- Policy: every SELECT/INSERT/UPDATE/DELETE is automatically scoped to current org.

ALTER TABLE nodes ENABLE ROW LEVEL SECURITY;

CREATE POLICY org_isolation ON nodes
    USING (org_id = current_setting('app.current_org_id')::uuid);

COMMIT;
