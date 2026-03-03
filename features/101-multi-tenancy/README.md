# T-101 — Multi-Tenancy

## Overview

Every entity in Infinite Brain is scoped to an organization. A personal user gets a personal org auto-created on registration. A company deploys one org (or multiple for business units). Data is strictly isolated at the database level via PostgreSQL Row-Level Security — no application bug can leak cross-org data.

---

## Why

Selling to companies requires hard data isolation. GDPR, SOC 2, and enterprise procurement all require demonstrable tenant separation. RLS enforces this at the engine level — not in application code that can have bugs.

---

## Organization Model

```sql
CREATE TABLE organizations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,      -- acme → acme.infinitebrain.io
    plan         TEXT NOT NULL DEFAULT 'personal', -- personal | pro | teams | enterprise
    max_members  INT,                       -- null = unlimited (enterprise)
    settings     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE org_members (
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member', -- owner | admin | editor | viewer
    invited_by UUID REFERENCES users(id),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX ON org_members (user_id); -- "which orgs does this user belong to?"
```

### Organization Settings (JSONB schema)

```jsonc
{
  "ai_provider": "claude",          // override default AI provider
  "mcp_server_url": "",             // custom MCP AI brain (T-097)
  "allowed_domains": ["acme.com"],  // restrict membership to email domains
  "require_mfa": true,              // enforce MFA for all members
  "data_retention_days": 365,       // auto-delete data older than N days
  "chunk_default_duration": 60,     // org-wide default chunk size
  "review_notifications": "digest"  // digest | realtime | off
}
```

---

## Org-Scoped Tables

Every table that holds user data gets `org_id`:

```sql
-- Migration: add org_id to all data tables
ALTER TABLE nodes           ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);
ALTER TABLE agent_memories  ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);
ALTER TABLE daily_plans     ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);
ALTER TABLE chunks          ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);
ALTER TABLE chunk_templates ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);
ALTER TABLE honeypot_hits   ADD COLUMN org_id UUID;  -- nullable: hits before auth have no org
ALTER TABLE personal_access_tokens ADD COLUMN org_id UUID NOT NULL REFERENCES organizations(id);

-- Indexes
CREATE INDEX ON nodes           (org_id);
CREATE INDEX ON agent_memories  (org_id);
CREATE INDEX ON daily_plans     (org_id, date);
```

---

## Row-Level Security

RLS is the hard guarantee. Even if application code has a bug that forgets to filter by org, the database refuses the query.

```sql
-- Set current org in transaction (called by auth middleware)
-- SET LOCAL app.current_org_id = '<uuid>';

-- Enable RLS on all data tables
ALTER TABLE nodes           ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_memories  ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_plans     ENABLE ROW LEVEL SECURITY;
ALTER TABLE chunks          ENABLE ROW LEVEL SECURITY;
ALTER TABLE chunk_templates ENABLE ROW LEVEL SECURITY;

-- Policy: every query is automatically scoped to current org
CREATE POLICY org_isolation ON nodes
    USING (org_id = current_setting('app.current_org_id')::uuid);

CREATE POLICY org_isolation ON agent_memories
    USING (org_id = current_setting('app.current_org_id')::uuid);

CREATE POLICY org_isolation ON daily_plans
    USING (org_id = current_setting('app.current_org_id')::uuid);

CREATE POLICY org_isolation ON chunks
    USING (org_id = current_setting('app.current_org_id')::uuid);

CREATE POLICY org_isolation ON chunk_templates
    USING (org_id = current_setting('app.current_org_id')::uuid);

-- Database role for application (NOT superuser — RLS applies)
-- Superuser bypasses RLS by default; app role does not
CREATE ROLE infinitebrain_app LOGIN PASSWORD '...';
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO infinitebrain_app;
```

### Setting org context in Go

```go
// internal/db/org_context.go

func WithOrgContext(ctx context.Context, db *pgxpool.Pool, orgID uuid.UUID, fn func(*pgxpool.Conn) error) error {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, "SET LOCAL app.current_org_id = $1", orgID.String())
    if err != nil {
        return fmt.Errorf("setting org context: %w", err)
    }

    return fn(conn)
}
```

Every database operation in a request goes through `WithOrgContext`. The RLS policy runs automatically — no `WHERE org_id = $1` needed in queries.

`★ Insight ─────────────────────────────────────`
Using a database role that is NOT superuser is critical. PostgreSQL superusers bypass RLS by default. The application should connect as `infinitebrain_app` (limited role), never as `postgres`. This ensures RLS is always enforced — no matter what query runs.
`─────────────────────────────────────────────────`

---

## Personal Org Auto-Creation

When a user registers or logs in for the first time:

```go
// internal/auth/user_sync.go

func (s *UserSyncer) provisionPersonalOrg(ctx context.Context, user *User) error {
    org := &Organization{
        Name: user.Name + "'s Brain",
        Slug: slugify(user.Email), // rian-example-com
        Plan: "personal",
    }
    org, err := s.orgRepo.Create(ctx, org)
    if err != nil {
        return err
    }
    return s.orgRepo.AddMember(ctx, org.ID, user.ID, RoleOwner)
}
```

The user's personal org is created with plan `personal`. They are the owner. They can later create or join additional orgs (team/enterprise).

---

## Organization API

```
POST   /api/v1/orgs                     Create org (plan upgrade flow)
GET    /api/v1/orgs/:slug               Get org details
PUT    /api/v1/orgs/:slug               Update org settings (admin only)
DELETE /api/v1/orgs/:slug               Delete org (owner only)

GET    /api/v1/orgs/:slug/members       List members
POST   /api/v1/orgs/:slug/members       Invite member (sends email)
PUT    /api/v1/orgs/:slug/members/:id   Update member role
DELETE /api/v1/orgs/:slug/members/:id   Remove member
```

---

## Subdomain Routing

Company deployments get their own subdomain: `acme.infinitebrain.io`

The HTTP server resolves org from subdomain before auth:

```go
// internal/middleware/org_resolver.go

func OrgResolver(orgRepo OrgRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            host := r.Host // acme.infinitebrain.io
            slug := extractSubdomain(host) // acme

            if slug == "" || slug == "www" || slug == "api" {
                // No subdomain — personal/default context resolved from token
                next.ServeHTTP(w, r)
                return
            }

            org, err := orgRepo.FindBySlug(r.Context(), slug)
            if err != nil {
                http.Error(w, "organization not found", http.StatusNotFound)
                return
            }

            ctx := context.WithValue(r.Context(), ctxOrgKey, org)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## Plan Limits

```go
// internal/org/limits.go

var planLimits = map[string]OrgLimits{
    "personal": {MaxMembers: 1,   MaxNodes: 10_000,  AICallsPerMonth: 500,   MaxProjects: 5},
    "pro":      {MaxMembers: 1,   MaxNodes: 0,       AICallsPerMonth: 0,     MaxProjects: 0},   // 0 = unlimited
    "teams":    {MaxMembers: 25,  MaxNodes: 0,       AICallsPerMonth: 0,     MaxProjects: 0},
    "enterprise":{MaxMembers: 0,  MaxNodes: 0,       AICallsPerMonth: 0,     MaxProjects: 0},
}
```

Limits enforced at service layer before operations. Exceeded limit returns `ErrPlanLimitReached` → 402 Payment Required.

---

## Acceptance Criteria

- [ ] `organizations` and `org_members` tables created via migration
- [ ] `org_id` column added to all data tables via migration
- [ ] RLS policies created for all data tables
- [ ] Application connects as `infinitebrain_app` role (not superuser)
- [ ] `WithOrgContext` sets `app.current_org_id` before every DB query
- [ ] Personal org auto-created on first user login
- [ ] Full CRUD for org management endpoints
- [ ] Member invite, role update, remove endpoints
- [ ] Subdomain middleware resolves org from host header
- [ ] Plan limits enforced at service layer
- [ ] Cross-org data access impossible even with valid token (RLS test)
- [ ] Integration test: two users in different orgs cannot see each other's nodes
- [ ] Integration test: superuser role NOT used by application (verify via pg_roles)
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL)
- T-007 / T-100 (Auth — org_id comes from token claims)
- T-028 (Knowledge graph — nodes table gets org_id)
- T-102 (RBAC — role column on org_members)

## Notes

- `slug` is immutable after creation — changing it would break subdomains and bookmarks
- Org deletion is a hard operation: cascade-deletes all nodes, memories, plans. Require explicit confirmation + 30-day grace period in production
- For self-hosted deployments with a single org, subdomain routing is disabled; org resolved from token only
