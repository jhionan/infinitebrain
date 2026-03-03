# T-098 — Security Hardening

## Overview

Security built in from the ground up, not bolted on. This task covers HTTP hardening, rate limiting, account lockout, input sanitization (prompt injection guard), secret scanning, and dependency vulnerability checks. PostgreSQL RLS (T-101) and RBAC (T-102) are the data layer — this task covers everything above it.

---

## HTTP Security Headers

Every response carries a strict security header set via Huma v2 middleware:

```go
// internal/middleware/security_headers.go

func SecurityHeaders() func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        h := ctx.Header()
        h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
        h.Set("X-Content-Type-Options", "nosniff")
        h.Set("X-Frame-Options", "DENY")
        h.Set("Referrer-Policy", "no-referrer")
        h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()")
        h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
        h.Set("X-Request-ID", requestIDFromContext(ctx.Context()))
        next(ctx)
    }
}
```

---

## TLS

- TLS 1.3 minimum in production (configured at reverse proxy / load balancer level)
- Local dev: HTTP allowed (`APP_ENV=development`)
- HSTS preload-ready header set on all responses

---

## Rate Limiting

Valkey-backed sliding window rate limiter. Applied per endpoint category:

```go
// internal/middleware/rate_limiter.go

type RateLimiter struct {
    valkey valkey.Client
}

// Sliding window: N requests per window duration, keyed by IP + path category
func (r *RateLimiter) Limit(key string, limit int, window time.Duration) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        ip := realIP(ctx.Header())
        k := fmt.Sprintf("rl:%s:%s", key, ip)

        count, err := r.valkey.Incr(ctx.Context(), k)
        if err == nil && count == 1 {
            r.valkey.Expire(ctx.Context(), k, window)
        }

        if count > int64(limit) {
            ctx.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
            ctx.SetStatus(http.StatusTooManyRequests)
            return
        }
        next(ctx)
    }
}
```

### Limits per endpoint

| Endpoint category | Limit | Window |
|---|---|---|
| `POST /auth/login` | 10 | 1 min |
| `POST /auth/register` | 5 | 1 hour |
| `POST /auth/refresh` | 30 | 1 min |
| `POST /api/v1/nodes` (capture) | 120 | 1 min |
| `POST /api/v1/memories/search` | 30 | 1 min |
| All other API endpoints | 300 | 1 min |
| MCP tool calls | 60 | 1 min |

---

## Account Lockout

After 5 consecutive failed login attempts, the account is locked for 15 minutes:

```go
// internal/auth/lockout.go

const (
    maxAttempts    = 5
    lockoutWindow  = 15 * time.Minute
    attemptsWindow = 10 * time.Minute
)

func (l *LockoutManager) Check(ctx context.Context, email string) error {
    key := fmt.Sprintf("lockout:%s", email)
    locked, _ := l.valkey.Get(ctx, key+"_locked")
    if locked != "" {
        ttl, _ := l.valkey.TTL(ctx, key+"_locked")
        return fmt.Errorf("%w: try again in %d minutes", ErrAccountLocked, int(ttl.Minutes())+1)
    }
    return nil
}

func (l *LockoutManager) RecordFailure(ctx context.Context, email string) error {
    key := fmt.Sprintf("lockout:%s", email)
    count, _ := l.valkey.Incr(ctx, key+"_attempts")
    if count == 1 {
        l.valkey.Expire(ctx, key+"_attempts", attemptsWindow)
    }
    if count >= maxAttempts {
        l.valkey.Set(ctx, key+"_locked", "1", lockoutWindow)
        l.valkey.Del(ctx, key+"_attempts")
    }
    return nil
}

func (l *LockoutManager) RecordSuccess(ctx context.Context, email string) {
    l.valkey.Del(ctx, fmt.Sprintf("lockout:%s_attempts", email))
}
```

---

## Request Size Limits

```go
// Applied at server startup
server := &http.Server{
    Handler: http.MaxBytesHandler(router, 10<<20), // 10 MB max body
}
```

Voice uploads (T-011) use a separate endpoint with a higher limit (50 MB) behind auth.

---

## CORS

```go
// internal/middleware/cors.go

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
    // Allowed origins from config: APP_CORS_ORIGINS=https://app.infinitebrain.io,https://acme.infinitebrain.io
    // Never wildcard in production
    // Credentials: true (JWT cookies if we add them later)
    // Methods: GET, POST, PUT, DELETE, OPTIONS
    // Headers: Authorization, Content-Type, X-Request-ID
}
```

---

## Prompt Injection Guard

All user-provided content that flows into AI prompts is sanitized before use. Prompt injection attacks attempt to override system prompts via captured content.

```go
// internal/ai/prompt_guard.go

var injectionPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+instructions?`),
    regexp.MustCompile(`(?i)you\s+are\s+now\s+`),
    regexp.MustCompile(`(?i)system\s*:\s*`),
    regexp.MustCompile(`(?i)assistant\s*:\s*`),
    regexp.MustCompile(`(?i)forget\s+everything`),
    regexp.MustCompile(`(?i)new\s+instructions?\s*:`),
    regexp.MustCompile(`(?i)jailbreak`),
    regexp.MustCompile(`(?i)<\s*\|?\s*im_start\s*\|?\s*>`),  // ChatML injection
}

type PromptGuard struct{ logger zerolog.Logger }

func (g *PromptGuard) Sanitize(content string) (string, bool) {
    for _, pattern := range injectionPatterns {
        if pattern.MatchString(content) {
            g.logger.Warn().Str("pattern", pattern.String()).Msg("prompt injection attempt detected")
            // Replace the offending section with [REDACTED] rather than rejecting outright
            content = pattern.ReplaceAllString(content, "[REDACTED]")
            return content, true // true = was modified
        }
    }
    return content, false
}
```

Every note content, task title, and voice transcription passes through `PromptGuard.Sanitize` before being inserted into any AI prompt.

---

## Dependency Vulnerability Scanning

In CI (`govulncheck` + `go mod verify`):

```yaml
# .github/workflows/ci.yml

- name: Vulnerability scan
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...

- name: Verify module integrity
  run: go mod verify

- name: Secret scan
  uses: trufflesecurity/trufflehog@main
  with:
    path: ./
    base: ${{ github.event.repository.default_branch }}
    head: HEAD
```

---

## Secret Detection

`.gitignore` patterns for sensitive files:
```
*.env
*.local
configs/local.*
zitadel-key.json
*.pem
*.key
*.p12
*.pfx
```

Pre-commit hook (`.githooks/pre-commit`):
```bash
#!/bin/sh
# Block commits with potential secrets
if git diff --cached --name-only | xargs grep -lE "(password|secret|api_key|private_key)\s*=\s*['\"][^'\"]{8,}" 2>/dev/null; then
    echo "Possible secret detected in staged files. Review before committing."
    exit 1
fi
```

---

## Security Response Headers Test

```go
// internal/middleware/security_headers_test.go

func TestSecurityHeaders(t *testing.T) {
    tests := []struct {
        header string
        want   string
    }{
        {"Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload"},
        {"X-Content-Type-Options", "nosniff"},
        {"X-Frame-Options", "DENY"},
        {"Referrer-Policy", "no-referrer"},
        {"X-Request-ID", ""},  // non-empty, any value
    }
    // httptest recorder + middleware + assert each header
}
```

---

## Acceptance Criteria

- [ ] Security headers middleware applied globally — verified via `curl -I` output in CI
- [ ] Rate limiter applied to all endpoint categories with correct limits
- [ ] Rate limiter returns 429 with `Retry-After` header
- [ ] Account lockout triggers after 5 failed logins; unlocks after 15 min
- [ ] `LockoutManager.RecordSuccess` clears attempt counter
- [ ] Request body capped at 10 MB (voice endpoint: 50 MB)
- [ ] CORS rejects origins not in `APP_CORS_ORIGINS`
- [ ] `PromptGuard.Sanitize` detects and redacts all 8 injection patterns
- [ ] Sanitized content is logged as a warning with pattern name
- [ ] `govulncheck` runs in CI and fails build on known vulnerabilities
- [ ] `go mod verify` runs in CI
- [ ] TruffleHog secret scan runs on every PR
- [ ] Pre-commit hook blocks obvious secret patterns
- [ ] Unit tests: `PromptGuard` — 8 patterns × clean + injected inputs = 16 cases
- [ ] Unit tests: `RateLimiter` — under limit, at limit, over limit, window reset
- [ ] Unit tests: `LockoutManager` — under threshold, at threshold, locked state, success reset
- [ ] Unit tests: `SecurityHeaders` — all headers present and correct
- [ ] Integration test: 6 failed logins → 7th returns 423 Locked
- [ ] **90% test coverage** on `internal/middleware/`, `internal/auth/`
- [ ] **100% test coverage** on `internal/ai/prompt_guard.go`

---

## Dependencies

- T-005 (Valkey — rate limiter + lockout storage)
- T-006 (HTTP server — middleware registration)
- T-020 (AI provider — prompt guard applied before all completions)

## Notes

- `realIP()` reads `X-Forwarded-For` / `X-Real-IP` only when `APP_BEHIND_PROXY=true` — prevents IP spoofing in direct-exposure deployments
- Rate limit keys include a hash of the IP — never store raw IPs in Valkey keys (GDPR)
- Prompt guard is a defence-in-depth measure — not a guarantee. AI providers also have their own guardrails.
- Injection patterns list should be versioned and expandable without code changes (future: load from config)
