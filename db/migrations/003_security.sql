-- Migration 003 — Security: honeypot hit log + blocked IP list.
-- Supports T-099 (honeypot endpoints + IP blocker).
-- Depends on: 001_core.sql

BEGIN;

-- honeypot_hits -----------------------------------------------------------------
-- Every call to a honeypot endpoint is recorded here.
-- ip is INET so PostgreSQL enforces valid address syntax.

CREATE TABLE honeypot_hits (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ip          INET        NOT NULL,
    path        TEXT        NOT NULL,
    method      TEXT        NOT NULL,
    user_agent  TEXT        NOT NULL DEFAULT '',
    headers     JSONB       NOT NULL DEFAULT '{}',
    body        TEXT        NOT NULL DEFAULT '',
    hit_count   INT         NOT NULL DEFAULT 1,
    blocked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookup: how many times has this IP hit the honeypot in the last 24 h?
CREATE INDEX honeypot_hits_ip_created_at_idx ON honeypot_hits (ip, created_at DESC);

-- Admin review: recent hits across all IPs.
CREATE INDEX honeypot_hits_created_at_idx ON honeypot_hits (created_at DESC);

-- blocked_ips -------------------------------------------------------------------
-- Fast-path block list. Valkey is the hot path; this table is the persistent
-- mirror for persistence across restarts and admin management.

CREATE TABLE blocked_ips (
    ip          INET        PRIMARY KEY,
    reason      TEXT        NOT NULL
                            CHECK (reason IN ('honeypot', 'manual', 'repeated_auth_failure')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Expire-sweeper: find rows whose block has lapsed.
CREATE INDEX blocked_ips_expires_at_idx ON blocked_ips (expires_at)
    WHERE expires_at IS NOT NULL;

COMMIT;
