# T-100 — Zitadel SSO Integration

## Overview

Integrate Zitadel as the identity provider for enterprise deployments. Zitadel handles authentication (OIDC/SAML), MFA, social login, and user management. Infinite Brain validates Zitadel-issued tokens and maps claims to internal users and organizations.

Personal/solo deployments continue to use internal JWT auth (T-007). Enterprise deployments switch to Zitadel via a single config flag.

---

## Why Zitadel

- Go-based, Apache 2.0, self-hostable
- OIDC + SAML 2.0 + LDAP/Active Directory out of the box
- Clean multi-tenancy via organizations
- Personal access tokens for API/MCP service accounts
- Same OIDC standard — also compatible with Keycloak, Okta, Azure AD, Google Workspace if a customer already has one

---

## Architecture

```
Client (bot / MCP / API)
    │
    ├── POST /auth/login ──→ redirect to Zitadel
    │                              │
    │                         User logs in
    │                         (SSO, MFA, social)
    │                              │
    │◄──────── OIDC token ─────────┘
    │
    ├── API request + Bearer token
    │
    ▼
Huma v2 middleware
    │
    ├── OIDCAuthenticator.Validate(token)
    │       │
    │       ├── Fetch JWKS from Zitadel discovery endpoint (cached)
    │       ├── Verify signature, issuer, audience, expiry
    │       └── Extract claims → UserID, OrgID, Email, Roles
    │
    └── Inject into request context → all handlers
```

---

## Authenticator Interface

```go
// internal/auth/authenticator.go

type Claims struct {
    UserID    uuid.UUID
    OrgID     uuid.UUID
    Email     string
    Roles     []string
    SessionID string
}

type Authenticator interface {
    Validate(ctx context.Context, token string) (*Claims, error)
}
```

Two implementations, selected at startup:

```go
// internal/auth/jwt_authenticator.go    — AUTH_MODE=jwt  (personal)
// internal/auth/oidc_authenticator.go   — AUTH_MODE=oidc (enterprise/Zitadel)
```

---

## OIDC Authenticator

```go
// internal/auth/oidc_authenticator.go

import (
    "github.com/coreos/go-oidc/v3/oidc/pkg/authorization"
    "github.com/coreos/go-oidc/v3/oidc/pkg/authorization/oauth"
    "github.com/coreos/go-oidc/v3/oidc/pkg/http/middleware"
)

type OIDCAuthenticator struct {
    authz      *authorization.Authorizer[*oauth.IntrospectionContext]
    audience   string
    orgClaim   string // Zitadel custom claim key for org_id
}

func NewOIDCAuthenticator(cfg config.OIDCConfig) (*OIDCAuthenticator, error) {
    authz, err := authorization.New(
        context.Background(),
        cfg.Issuer,
        oauth.DefaultAuthorization(cfg.KeyPath),
    )
    if err != nil {
        return nil, fmt.Errorf("oidc init: %w", err)
    }
    return &OIDCAuthenticator{authz: authz, audience: cfg.Audience}, nil
}

func (a *OIDCAuthenticator) Validate(ctx context.Context, token string) (*Claims, error) {
    // Introspect token via Zitadel
    authCtx, err := a.authz.CheckAuthorization(ctx, token)
    if err != nil {
        return nil, ErrUnauthorized
    }

    userID, err := uuid.Parse(authCtx.UserID())
    if err != nil {
        return nil, fmt.Errorf("invalid user_id claim: %w", err)
    }

    orgID := a.extractOrgID(authCtx)

    return &Claims{
        UserID:    userID,
        OrgID:     orgID,
        Email:     authCtx.Email,
        Roles:     authCtx.Roles(),
        SessionID: authCtx.SessionID,
    }, nil
}
```

---

## User Sync

Zitadel is the source of truth for identity. Infinite Brain maintains a local `users` table for application data (preferences, settings). Users are synced on first token validation:

```go
// internal/auth/user_sync.go

func (s *UserSyncer) SyncOnLogin(ctx context.Context, claims *Claims) (*User, error) {
    user, err := s.repo.FindByExternalID(ctx, claims.UserID.String())
    if errors.Is(err, ErrNotFound) {
        // First login — provision user + personal org
        return s.repo.CreateFromClaims(ctx, claims)
    }
    if err != nil {
        return nil, err
    }
    // Update email if changed in Zitadel
    if user.Email != claims.Email {
        return s.repo.UpdateEmail(ctx, user.ID, claims.Email)
    }
    return user, nil
}
```

```sql
ALTER TABLE users ADD COLUMN external_id TEXT UNIQUE; -- Zitadel user ID (sub claim)
ALTER TABLE users ADD COLUMN auth_provider TEXT NOT NULL DEFAULT 'internal'; -- 'internal' | 'zitadel'
CREATE INDEX ON users (external_id);
```

---

## Personal Access Tokens (for API / MCP)

Service accounts and MCP clients need long-lived tokens without browser-based login. Zitadel supports personal access tokens natively. In Infinite Brain:

```
POST /api/v1/tokens
Body: { "name": "Claude Code MCP", "expires_at": "2027-01-01" }
Response: { "token": "ibpat_...", "id": "..." }
```

Personal access tokens are validated like JWTs but via a separate fast-path:

```sql
CREATE TABLE personal_access_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,  -- argon2id hash
    last_used_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Token format: `ibpat_` prefix + 32 random bytes base64url. The prefix lets anyone identify it as an Infinite Brain token (like GitHub's `ghp_` prefix).

---

## Docker Compose (local dev)

```yaml
# docker-compose.yml addition
zitadel:
  image: ghcr.io/zitadel/zitadel:latest
  command: start-from-init --masterkey "MasterkeyNeedsToHave32Characters" --tlsMode disabled
  environment:
    ZITADEL_DATABASE_POSTGRES_HOST: postgres
    ZITADEL_DATABASE_POSTGRES_PORT: 5432
    ZITADEL_DATABASE_POSTGRES_DATABASE: zitadel
    ZITADEL_DATABASE_POSTGRES_USER_USERNAME: zitadel
    ZITADEL_DATABASE_POSTGRES_USER_PASSWORD: zitadel
    ZITADEL_DATABASE_POSTGRES_USER_SSL_MODE: disable
    ZITADEL_EXTERNALDOMAIN: localhost
    ZITADEL_EXTERNALPORT: 8080
    ZITADEL_EXTERNALSECURE: false
    ZITADEL_FIRSTINSTANCE_ORG_HUMAN_USERNAME: admin@infinitebrain.local
    ZITADEL_FIRSTINSTANCE_ORG_HUMAN_PASSWORD: Password1!
  ports:
    - "8080:8080"
  depends_on:
    postgres:
      condition: service_healthy
```

Zitadel uses the same PostgreSQL instance — no extra DB service needed.

---

## Configuration

```bash
# configs/example.env

AUTH_MODE=oidc                              # jwt | oidc
AUTH_OIDC_ISSUER=http://localhost:8080      # Zitadel base URL
AUTH_OIDC_AUDIENCE=infinite-brain-api       # Zitadel API application client ID
AUTH_OIDC_KEY_PATH=./zitadel-key.json      # Zitadel introspection key file
```

---

## Library

```
github.com/coreos/go-oidc/v3
golang.org/x/oauth2
```

Standard OIDC library — not Zitadel-specific. Works with Zitadel, Keycloak, Okta, Azure AD, Google Workspace. Our code never knows which identity provider is running — only config changes. JWKS keys fetched from the discovery endpoint on startup and cached with auto-rotation:

```go
provider, _ := oidc.NewProvider(ctx, cfg.OIDCIssuer)
verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDCAudience})
idToken, err := verifier.Verify(ctx, rawToken)
```

---

## Acceptance Criteria

- [ ] `OIDCAuthenticator` validates Zitadel-issued JWTs (signature, issuer, audience, expiry)
- [ ] `JWTAuthenticator` (T-007) and `OIDCAuthenticator` both implement `Authenticator` interface
- [ ] `AUTH_MODE=jwt` uses internal auth; `AUTH_MODE=oidc` uses Zitadel
- [ ] First login provisions user + personal org automatically
- [ ] Email updates in Zitadel sync to local users table on next login
- [ ] Personal access tokens: create, list, revoke endpoints
- [ ] Personal access tokens validated in auth middleware (fast-path, no Zitadel call)
- [ ] Zitadel added to docker-compose.yml for local dev
- [ ] `make dev` brings up Zitadel alongside PostgreSQL and Valkey
- [ ] Makefile target: `make zitadel-setup` — creates application, API, and test user via Zitadel management API
- [ ] Unit tests for `OIDCAuthenticator.Validate` with mock JWKS server
- [ ] Integration test: full login flow with local Zitadel instance
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL)
- T-007 (Auth — interface being extended)
- T-101 (Multi-tenancy — org_id in claims)

## Notes

- Zitadel runs on port 8080 locally; REST API on 8090; MCP server on 8091 — no conflicts
- The `ibpat_` token prefix is intentional — scanners like GitGuardian and TruffleHog can be configured to detect and alert on exposed Infinite Brain tokens (security by design)
- For Zitadel cloud (managed): just change `AUTH_OIDC_ISSUER` — no code change needed
- Compatible with any OIDC-compliant provider (Keycloak, Okta, Azure AD) — customer brings their own
