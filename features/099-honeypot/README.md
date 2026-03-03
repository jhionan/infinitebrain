# T-099 — Honeypot Endpoints

## Overview

A set of fake endpoints that legitimate users and clients never call. When hit, the source IP is flagged, logged, and progressively blocked. Automated scanners, vulnerability probers, and bots expose themselves immediately. The responses look real — wasting attacker time and triggering their next automated step against fake data.

---

## Why

Most APIs get probed constantly. Infinite Brain handles personal and company knowledge — a high-value target. A honeypot adds active threat detection with zero false positives: no legitimate client should ever call these endpoints.

This is a strong portfolio signal — it shows you think about security beyond "add auth middleware".

---

## Honeypot Endpoints

All endpoints registered before auth middleware — they are intentionally accessible unauthenticated.

| Path | Method | Mimics | Fake Response |
|---|---|---|---|
| `/.env` | GET | Exposed env file | Fake key=value file with plausible-looking fake secrets |
| `/.git/config` | GET | Exposed git config | Fake git config with remote URL |
| `/wp-login.php` | GET/POST | WordPress login | HTML login page (WordPress-looking) |
| `/phpinfo.php` | GET | PHP info page | Minimal PHP info HTML |
| `/api/v1/admin` | GET | Admin panel | JSON `{ "users": [], "stats": {} }` |
| `/api/v1/internal/debug` | GET | Debug dump | Fake memory/config dump JSON |
| `/api/v1/users/export-all` | GET | Data export | Fake CSV headers, empty rows |
| `/api/v1/internal/config/reset` | POST | Config reset | `{ "status": "reset scheduled" }` |

Responses use realistic content types and status codes (200, not 404) — scanners move on when they get 404s. A 200 with plausible content keeps them engaged and generates more hits for tracking.

---

## Schema

```sql
CREATE TABLE honeypot_hits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip          INET NOT NULL,
    path        TEXT NOT NULL,
    method      TEXT NOT NULL,
    user_agent  TEXT,
    headers     JSONB NOT NULL DEFAULT '{}',
    body        TEXT,
    hit_count   INT NOT NULL DEFAULT 1,     -- for this IP today
    blocked_at  TIMESTAMPTZ,                -- when auto-block was triggered
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON honeypot_hits (ip, created_at DESC);
CREATE INDEX ON honeypot_hits (created_at DESC);  -- for admin review
```

```sql
-- Blocked IPs (checked before all endpoints, including honeypot)
-- Stored in Valkey for fast lookup, mirrored here for persistence
CREATE TABLE blocked_ips (
    ip          INET PRIMARY KEY,
    reason      TEXT NOT NULL,  -- 'honeypot' | 'manual' | 'repeated_auth_failure'
    expires_at  TIMESTAMPTZ,    -- null = permanent
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## Auto-Block Logic

```go
// internal/security/honeypot.go

const (
    hitsBefore24hBlock   = 2   // 2 honeypot hits → 24h block
    hitsBefore7dBlock    = 5   // 5 hits → 7d block
    hitsPermanentBlock   = 10  // 10 hits → permanent block
)

func (h *HoneypotHandler) Handle(path string) huma.Handler {
    return func(ctx huma.Context, _ *struct{}) (*HoneypotOutput, error) {
        ip := realIP(ctx.Header())

        hit := &HoneypotHit{
            IP:        ip,
            Path:      path,
            Method:    ctx.Method(),
            UserAgent: ctx.Header().Get("User-Agent"),
            Headers:   headersToJSON(ctx.Header()),
            Body:      readBody(ctx),
        }

        h.repo.Record(ctx.Context(), hit)
        h.logger.Warn().
            Str("ip", ip).
            Str("path", path).
            Str("user_agent", hit.UserAgent).
            Msg("honeypot hit")

        h.maybeBlock(ctx.Context(), ip)

        // Return realistic fake response — never 404
        return fakeResponse(path), nil
    }
}

func (h *HoneypotHandler) maybeBlock(ctx context.Context, ip string) {
    count, _ := h.repo.HitCountLast24h(ctx, ip)

    switch {
    case count >= hitsPermanentBlock:
        h.block(ctx, ip, 0, "honeypot:permanent")      // 0 = no expiry
        h.logger.Error().Str("ip", ip).Msg("permanent IP block triggered")
    case count >= hitsBefore7dBlock:
        h.block(ctx, ip, 7*24*time.Hour, "honeypot:7d")
    case count >= hitsBefore24hBlock:
        h.block(ctx, ip, 24*time.Hour, "honeypot:24h")
    }
}

func (h *HoneypotHandler) block(ctx context.Context, ip string, duration time.Duration, reason string) {
    // Write to Valkey (fast path for IP block check middleware)
    key := fmt.Sprintf("blocked_ip:%s", ip)
    if duration == 0 {
        h.valkey.Set(ctx, key, reason, 0) // no expiry
    } else {
        h.valkey.Set(ctx, key, reason, duration)
    }
    // Mirror to DB for persistence and admin review
    h.repo.BlockIP(ctx, ip, reason, duration)
}
```

---

## IP Block Middleware

Runs before everything — before auth, before honeypot:

```go
// internal/middleware/ip_block.go

func IPBlocker(valkey valkey.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := realIP(r.Header)
            key := fmt.Sprintf("blocked_ip:%s", ip)

            reason, err := valkey.Get(r.Context(), key)
            if err == nil && reason != "" {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

Valkey lookup: < 1ms. No DB hit on the hot path.

---

## Fake Responses

```go
// internal/security/fake_responses.go

func fakeResponse(path string) *HoneypotOutput {
    switch path {
    case "/.env":
        return &HoneypotOutput{
            ContentType: "text/plain",
            Body: strings.Join([]string{
                "APP_ENV=production",
                "DATABASE_URL=postgres://admin:p@ssw0rd!@db.internal:5432/infinitebrain",
                "ANTHROPIC_API_KEY=sk-ant-api03-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
                "JWT_SECRET=aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcde",
                "REDIS_URL=redis://:s3cr3t@cache.internal:6379/0",
            }, "\n"),
        }
    case "/.git/config":
        return &HoneypotOutput{
            ContentType: "text/plain",
            Body: "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n[remote \"origin\"]\n\turl = git@github.com:infinitebrain/infinite-brain-private.git\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n",
        }
    case "/wp-login.php":
        return &HoneypotOutput{
            ContentType: "text/html",
            Body:        wpLoginHTML, // static HTML mimicking WP login
        }
    case "/api/v1/admin":
        return &HoneypotOutput{
            ContentType: "application/json",
            Body:        `{"users":[],"total":0,"plan":"enterprise","version":"3.2.1"}`,
        }
    // ... other paths
    }
    return &HoneypotOutput{ContentType: "application/json", Body: `{"status":"ok"}`}
}
```

The fake `.env` API key passes GitGuardian's format check — attackers try to use it. That attempt becomes another signal.

---

## Admin Endpoints

```
GET /api/v1/admin/honeypot/hits         Recent hits (admin only, via RBAC)
GET /api/v1/admin/honeypot/blocked      Currently blocked IPs
DELETE /api/v1/admin/honeypot/blocked/:ip  Unblock an IP (false positive)
```

---

## Acceptance Criteria

- [ ] All 8 honeypot endpoints registered before auth middleware
- [ ] Every hit logged to `honeypot_hits` with IP, path, user agent, headers
- [ ] Auto-block at 2 hits (24h), 5 hits (7d), 10 hits (permanent)
- [ ] Blocked IPs written to Valkey immediately (< 1ms lookup)
- [ ] Blocked IPs mirrored to `blocked_ips` table
- [ ] `IPBlocker` middleware runs before all other middleware
- [ ] Blocked IP returns 403 on all endpoints including non-honeypot
- [ ] Admin can unblock an IP via API (removes from Valkey + DB)
- [ ] Fake `.env` response contains realistic-looking fake credentials
- [ ] `/wp-login.php` returns HTML with 200 status (not 404)
- [ ] Unit tests: `maybeBlock` — all 4 threshold scenarios
- [ ] Unit tests: `fakeResponse` — correct content type per path
- [ ] Unit tests: `IPBlocker` — blocked IP returns 403, clean IP passes through
- [ ] Integration test: hit honeypot 2× → 3rd regular API call returns 403
- [ ] Integration test: admin unblock → subsequent requests pass
- [ ] **90% test coverage** on `internal/security/`

---

## Dependencies

- T-005 (Valkey — block list fast lookup)
- T-004 (PostgreSQL — hit logging + block persistence)
- T-006 (HTTP server — middleware registration order matters)
- T-102 (RBAC — admin endpoints require owner/admin role)

## Notes

- Honeypot endpoints must be registered AFTER `IPBlocker` middleware but BEFORE auth middleware
- Never log honeypot hits to the general request log — they go to a dedicated security log channel
- `realIP()` must handle `X-Forwarded-For` chains correctly — take the leftmost non-private IP
- Consider adding a Slack/email alert for permanent blocks — those are high-confidence threat actors
