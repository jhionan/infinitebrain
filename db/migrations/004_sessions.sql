-- Migration 004 — Auth sessions (refresh tokens).
-- Refresh tokens are stored as SHA-256 hashes — never plaintext.
-- expires_at enforced in application layer AND in FindSessionByTokenHash query.

BEGIN;

CREATE TABLE sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    org_id       UUID        NOT NULL REFERENCES orgs(id)   ON DELETE CASCADE,
    token_hash   TEXT        NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX sessions_user_id_idx    ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

COMMIT;
