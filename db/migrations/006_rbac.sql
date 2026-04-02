-- db/migrations/006_rbac.sql
-- Migration 006 — RBAC: audit_log + org_invites tables.
-- Generated from db/schema/006_rbac.hcl

BEGIN;

-- audit_log: append-only audit trail ----------------------------------------

CREATE TABLE audit_log (
    id          UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES orgs(id)  ON DELETE CASCADE,
    actor_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action      TEXT        NOT NULL,
    target_type TEXT,
    target_id   UUID,
    before      JSONB,
    after       JSONB,
    ip          INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_org_id_created_at_idx ON audit_log (org_id, created_at);
CREATE INDEX audit_log_actor_id_idx           ON audit_log (actor_id);

-- org_invites: pending invitations to join an org ----------------------------

CREATE TABLE org_invites (
    id          UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES orgs(id)  ON DELETE CASCADE,
    email       TEXT        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'editor'
                            CHECK (role IN ('owner', 'admin', 'editor', 'viewer', 'member')),
    invited_by  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '7 days'),
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX org_invites_org_id_idx ON org_invites (org_id);

COMMIT;
