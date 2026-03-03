# T-104 — Compliance: SOC2 + HIPAA + EU AI Act + GDPR

## Overview

Implement the technical controls required for:
- **SOC2 Type II** (AICPA Trust Services Criteria) — required by enterprise buyers
- **HIPAA** (45 CFR Parts 160/164) — required for PHI handling
- **EU AI Act** (Regulation 2024/1689) — required for AI systems used in the EU (see T-154)
- **GDPR** (Regulation 2016/679) — required for EU personal data

This is not a checkbox exercise — these controls make the system genuinely more secure and
trustworthy. Enterprise customers, healthcare-adjacent companies, and any B2B sale will require
evidence of these controls.

This feature adds the compliance infrastructure that all other features build on top of.

**EU AI Act compliance is owned by T-154.** T-104 owns the data/security layer (HIPAA, SOC2, GDPR).
The audit log and encryption infrastructure here serves all four frameworks.

---

## Why Both

**SOC2** (AICPA Trust Services Criteria) — required by enterprise buyers. Auditors will ask
for evidence that your security controls (CC6–CC9) are designed and operating effectively.
SOC2 Type II means the controls were audited over a period of time, not just a point-in-time.

**HIPAA** (45 CFR Part 164) — required if any user stores health information (medications,
doctor appointments, therapy notes, fitness data). For Infinite Brain, this is likely: our
ADHD users will capture health-adjacent data. A BAA-capable product opens the Teams/Enterprise
tier to healthcare companies (clinics, ADHD coaching practices, health-tech startups).

Both share the same technical foundation. Building them together costs ~20% more than just
SOC2 but doubles the addressable enterprise market.

---

## Compliance Scope

### What constitutes PHI in Infinite Brain

PHI (Protected Health Information) is any data that:
- Relates to health, treatment, or payment for healthcare
- Can identify an individual

In our schema, this applies to:
- Node content tagged with health-related topics (medications, diagnoses, symptoms)
- Voice note transcriptions mentioning health information
- Any node with `metadata.is_phi = true`

We implement **field-level encryption** for all PHI and **data classification** to track it.

---

## 1. Data Classification

Every column that holds user content is classified. Classification lives in the schema and
is enforced at the application layer.

```go
// pkg/compliance/classification.go

type DataClass int

const (
    ClassPublic      DataClass = iota // no restrictions
    ClassInternal                     // org-internal only
    ClassConfidential                 // PII — encrypted at rest
    ClassRestricted                   // PHI — field-level encrypted, audit every access
)
```

Annotated in sqlc query comments and enforced in repository layer:

```sql
-- name: GetNodeContent :one
-- data_class: confidential
-- phi_possible: true
SELECT id, content, metadata FROM nodes WHERE id = $1 AND org_id = $2;
```

The repository wrapper checks: if `metadata->>'is_phi' = 'true'`, route through the
`PHIAccessor` which logs the access and decrypts inline.

---

## 2. Field-Level Encryption (HIPAA § 164.312(a)(2)(iv))

TLS protects data in transit. Encryption at rest protects the disk if storage is stolen.
Field-level encryption protects data even if the database is compromised — an attacker
with a raw dump of the `nodes` table cannot read PHI without the encryption key.

### Algorithm: AES-256-GCM

```go
// pkg/crypto/field_encryption.go

type FieldEncryptor struct {
    keyring KeyRing  // supports multiple keys for rotation
}

// Encrypt returns ciphertext in the format: base64(keyID || nonce || ciphertext || tag)
// The keyID prefix allows key rotation without re-encrypting all data immediately.
func (e *FieldEncryptor) Encrypt(plaintext []byte) (string, error)

// Decrypt resolves the keyID from the ciphertext prefix, fetches the key, and decrypts.
func (e *FieldEncryptor) Decrypt(ciphertext string) ([]byte, error)
```

### Key Management

```go
// pkg/crypto/keyring.go

type KeyRing interface {
    // ActiveKey returns the key used for new encryptions.
    ActiveKey(ctx context.Context) (*EncryptionKey, error)

    // KeyByID retrieves a specific key (for decryption of existing data).
    KeyByID(ctx context.Context, id string) (*EncryptionKey, error)

    // RotateKey generates a new active key. Old keys remain for decryption.
    RotateKey(ctx context.Context) (*EncryptionKey, error)
}

type EncryptionKey struct {
    ID        string
    Material  []byte // 32 bytes for AES-256
    CreatedAt time.Time
    ExpiresAt *time.Time
}
```

Two implementations:
- `EnvKeyRing` — key material in environment variable (dev/small deployments)
- `OpenBaoKeyRing` — key material in OpenBao (production); auto-rotation built in

**OpenBao** is the open source fork of HashiCorp Vault (MPL-2.0), maintained by the OpenBao
community after HashiCorp's license change. Drop-in API compatible. Self-hostable.

Keys are **per-tenant** in the enterprise tier: a data breach of one org's key cannot
expose another org's PHI. The `org_id` is included in the AAD (additional authenticated data)
of each AES-GCM encryption, binding the ciphertext to its owner.

### Schema

```sql
-- Encryption keys table (keys themselves are stored encrypted by a master key)
CREATE TABLE encryption_keys (
    id           TEXT PRIMARY KEY,       -- random key ID embedded in ciphertext
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key_material TEXT NOT NULL,          -- encrypted by master key (KMS/Vault)
    algorithm    TEXT NOT NULL DEFAULT 'AES-256-GCM',
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at   TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ
);

CREATE INDEX ON encryption_keys (org_id, is_active);
```

### PHI Column Marker

```sql
-- nodes table: content column stores encrypted ciphertext when is_phi = true
ALTER TABLE nodes ADD COLUMN content_encrypted BOOLEAN NOT NULL DEFAULT false;

-- When content_encrypted = true, the content column holds the AES-256-GCM ciphertext.
-- The repository layer transparently encrypts on write and decrypts on read.
```

---

## 3. Immutable Audit Log (SOC2 CC6.1, CC7.2 / HIPAA § 164.312(b))

The audit log from T-102 (RBAC) is extended to be tamper-evident and covers all
PHI access, not just permission changes.

### What Gets Logged

| Event | SOC2 | HIPAA |
|---|---|---|
| User login / logout | CC6.1 | § 164.312(d) |
| Failed login attempts | CC6.1 | § 164.312(d) |
| PHI read access | CC6.1 | § 164.312(b) |
| PHI create / update / delete | CC6.1 | § 164.312(b) |
| Permission changes | CC6.3 | — |
| Data export | CC6.7 | § 164.312(b) |
| Data deletion | CC6.7 | § 164.308(a)(3) |
| Key rotation | CC6.8 | § 164.312(a)(2)(iv) |
| Config changes | CC8.1 | — |
| API key created / revoked | CC6.2 | — |

### Tamper-Evidence via Hash Chain

Each audit log entry includes the SHA-256 hash of the previous entry. This creates a
chain: modifying or deleting any historical entry breaks the chain, which is detectable.

```go
// internal/audit/log.go

type AuditEvent struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    ActorID      uuid.UUID       // user who performed the action
    ActorType    string          // "user" | "service_account" | "system"
    Action       string          // "node.read" | "node.update" | "auth.login" etc.
    ResourceType string
    ResourceID   uuid.UUID
    IPAddress    string
    UserAgent    string
    Outcome      string          // "success" | "denied" | "error"
    PHIAccessed  bool
    Metadata     map[string]any
    PrevHash     string          // SHA-256 of previous entry (hash chain)
    Hash         string          // SHA-256 of this entry's content + PrevHash
    CreatedAt    time.Time
}
```

```sql
CREATE TABLE audit_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL,
    actor_id      UUID NOT NULL,
    actor_type    TEXT NOT NULL,
    action        TEXT NOT NULL,
    resource_type TEXT,
    resource_id   UUID,
    ip_address    INET,
    user_agent    TEXT,
    outcome       TEXT NOT NULL,
    phi_accessed  BOOLEAN NOT NULL DEFAULT false,
    metadata      JSONB DEFAULT '{}',
    prev_hash     TEXT NOT NULL,
    hash          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only: no UPDATE or DELETE permissions granted to application role
-- REVOKE UPDATE, DELETE ON audit_log FROM infinitebrain_app;

CREATE INDEX ON audit_log (org_id, created_at DESC);
CREATE INDEX ON audit_log (org_id, phi_accessed) WHERE phi_accessed = true;
CREATE INDEX ON audit_log (actor_id, created_at DESC);
```

### Audit Service

```go
// internal/audit/service.go

type AuditLogger interface {
    Log(ctx context.Context, event AuditEvent) error

    // VerifyChain checks the hash chain integrity for an org's audit log.
    // Returns the first broken entry if tampered.
    VerifyChain(ctx context.Context, orgID uuid.UUID) (*ChainVerificationResult, error)
}
```

Called from middleware — handlers never call the audit logger directly:

```go
// internal/middleware/audit.go

func AuditMiddleware(auditor audit.AuditLogger) connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            resp, err := next(ctx, req)
            outcome := "success"
            if err != nil {
                outcome = "error"
            }
            _ = auditor.Log(ctx, audit.AuditEvent{
                Action:   req.Spec().Procedure,
                Outcome:  outcome,
                // ... extracted from context
            })
            return resp, err
        }
    }
}
```

---

## 4. Automatic Session Timeout (HIPAA § 164.312(a)(2)(iii))

Sessions expire after inactivity. Configurable per org:

```go
// internal/auth/session_policy.go

type SessionPolicy struct {
    IdleTimeout    time.Duration // default: 15 minutes for PHI orgs, 60 min otherwise
    AbsoluteTimeout time.Duration // default: 8 hours — no matter what
    RequireMFAFor  []string      // e.g. ["phi_access", "admin_actions"]
}
```

JWT tokens carry an `iat` (issued-at) and `exp` (expiry). The refresh token endpoint
enforces `IdleTimeout` by rejecting tokens where `now - last_activity > idle_timeout`.

```sql
ALTER TABLE sessions ADD COLUMN last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE sessions ADD COLUMN phi_accessed_at  TIMESTAMPTZ; -- triggers shorter timeout
```

---

## 5. Minimum Necessary Access (HIPAA § 164.502(b))

Users and service accounts should only access the PHI they need for their specific task.
Applied in three places:

**1. Query scoping** — PHI columns are excluded from list queries by default:
```sql
-- name: ListNodes :many
-- PHI content is NOT returned in list queries. Fetch individually with GetNode.
SELECT id, title, type, para, tags, created_at FROM nodes WHERE org_id = $1;
```

**2. Role enforcement** — `viewer` role cannot access raw PHI content:
```go
// internal/rbac/permissions.go
PermissionReadPHI    Permission = "phi:read"    // editor+ only
PermissionExportData Permission = "data:export" // admin+ only
```

**3. Service account scoping** — personal access tokens (T-100) can be scoped:
```sql
ALTER TABLE personal_access_tokens ADD COLUMN scopes TEXT[] NOT NULL DEFAULT '{}';
-- e.g. scopes = ['nodes:read', 'tasks:write'] — no PHI access
```

---

## 6. Encryption in Transit (HIPAA § 164.312(e)(2)(ii) / SOC2 CC6.7)

- TLS 1.3 minimum — no TLS 1.2 or below
- HSTS header with `max-age=63072000; includeSubDomains; preload`
- Certificate pinning documentation for mobile SDK clients
- Internal service communication (gRPC) also requires mTLS in production

```go
// cmd/server/main.go

tlsCfg := &tls.Config{
    MinVersion: tls.VersionTLS13,
    CurvePreferences: []tls.CurveID{
        tls.X25519,
        tls.CurveP256,
    },
}
```

---

## 7. Data Retention and Right to Erasure (SOC2 CC6.5 / HIPAA § 164.530(j))

### Retention Policies

```sql
CREATE TABLE data_retention_policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id),
    data_type   TEXT NOT NULL,   -- 'nodes' | 'audit_log' | 'agent_memories' | 'sessions'
    retain_days INT NOT NULL,    -- 0 = indefinite
    legal_hold  BOOLEAN NOT NULL DEFAULT false, -- overrides retention during legal proceedings
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Default policies inserted on org creation:
-- nodes: 0 (indefinite, controlled by user)
-- audit_log: 2555 (7 years — SOC2 and HIPAA require 6 years minimum)
-- sessions: 90 days
-- agent_memories: 365 days (overridable)
```

### Right to Erasure API

```
DELETE /api/v1/account
Body: { "confirm": "DELETE MY ACCOUNT", "reason": "user_request" }
```

This triggers a `DataErasureJob` (River) that:
1. Hard-deletes all user data (CASCADE handles DB)
2. Purges Valkey keys for the user
3. Revokes all active sessions and tokens
4. Removes S3 files
5. Logs the erasure event in the audit log (audit log entry is kept — erasure of audit logs
   is not permitted under HIPAA/SOC2; the log records *that* erasure happened, not the content)
6. Sends erasure confirmation email

**Audit log is never deleted** — it records the erasure event itself. This is by design.

---

## 8. Breach Detection and Notification (HIPAA § 164.400 / SOC2 CC7.3)

### Anomaly Detection

```go
// internal/security/anomaly.go

type AnomalyDetector struct {
    auditor  audit.AuditLogger
    alerter  Alerter
    valkey   valkey.Client
}

// Patterns that trigger alerts:
// - Same user accessing PHI from 2+ countries in < 1 hour
// - > 100 PHI reads in < 5 minutes (bulk exfiltration pattern)
// - Login from new device/IP after long dormancy
// - Service account accessing PHI (service accounts should never need PHI)
func (d *AnomalyDetector) Analyze(ctx context.Context, event audit.AuditEvent) error
```

### Breach Notification Timeline

HIPAA requires notification within 60 days of discovery. The system supports this with:

```sql
CREATE TABLE security_incidents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID REFERENCES organizations(id),
    severity         TEXT NOT NULL,  -- 'low' | 'medium' | 'high' | 'critical'
    type             TEXT NOT NULL,  -- 'unauthorized_access' | 'data_export' | 'anomaly' etc.
    discovered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    contained_at     TIMESTAMPTZ,
    notified_at      TIMESTAMPTZ,   -- when affected parties were notified
    phi_involved     BOOLEAN NOT NULL DEFAULT false,
    affected_records INT,
    description      TEXT NOT NULL,
    remediation      TEXT,
    created_by       UUID NOT NULL   -- system or admin user ID
);
```

---

## 9. BAA Support (Business Associate Agreement)

For HIPAA, any vendor processing PHI on behalf of a covered entity must sign a BAA.
The system records BAA acceptance:

```sql
CREATE TABLE baa_agreements (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id),
    accepted_by  UUID NOT NULL REFERENCES users(id),
    accepted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    version      TEXT NOT NULL,     -- BAA document version
    ip_address   INET NOT NULL
);
```

An org without a signed BAA cannot enable the `phi_mode` flag. Attempting to store PHI
without BAA acceptance returns a `423 Locked` with `code: BAA_REQUIRED`.

---

## 10. Compliance Dashboard (Admin API)

Endpoints for compliance officers and auditors:

```
GET  /api/v1/admin/compliance/audit-log?from=&to=&phi_only=true
GET  /api/v1/admin/compliance/audit-log/verify-chain
GET  /api/v1/admin/compliance/data-access-report?user_id=&from=&to=
GET  /api/v1/admin/compliance/phi-inventory
GET  /api/v1/admin/compliance/key-rotation-status
POST /api/v1/admin/compliance/rotate-keys
GET  /api/v1/admin/compliance/incidents
POST /api/v1/admin/compliance/incidents
```

---

## 11. Password Hashing: Salt + Pepper (OWASP 2026)

argon2id alone generates a unique salt per password and is resistant to GPU brute-force.
Adding a pepper provides a second layer: even if the entire database is stolen, hashes
cannot be cracked without the pepper value, which never touches the database.

```
stored_hash = argon2id( password + pepper , random_salt , argon2id_params )
```

- **Salt**: random, unique per password, stored in the hash string (argon2id embeds it)
- **Pepper**: a global secret loaded from OpenBao at startup — never stored in the DB

```go
// pkg/crypto/password.go

type PasswordHasher struct {
    pepper []byte // loaded from OpenBao / env var at startup; never from DB
}

func (h *PasswordHasher) Hash(password string) (string, error) {
    pepperedInput := append([]byte(password), h.pepper...)
    return argon2id.CreateHash(string(pepperedInput), argon2id.DefaultParams)
}

func (h *PasswordHasher) Verify(password, hash string) (bool, error) {
    pepperedInput := append([]byte(password), h.pepper...)
    return argon2id.ComparePasswordAndHash(string(pepperedInput), hash)
}
```

### Pepper Rotation

When the pepper is rotated, all existing hashes are invalidated. The re-hashing process:
1. On next login, verify the password against the old pepper
2. If valid, immediately re-hash with the new pepper and update the stored hash
3. After a migration window, force re-authentication for accounts not yet rotated

```sql
ALTER TABLE users ADD COLUMN pepper_version INT NOT NULL DEFAULT 1;
-- Tracks which pepper version was used — enables gradual rotation
```

---

## 12. Secret Auto-Rotation (SOC2 CC6.8 / HIPAA § 164.312(a)(2)(iv))

All secrets rotate automatically. Manual rotation is a SOC2 finding — automation is evidence.

### Rotation Schedule

| Secret | Rotation Interval | Method |
|---|---|---|
| Encryption keys (per-tenant) | 90 days | OpenBao lease TTL → `KeyRing.RotateKey()` |
| Password pepper | 180 days | OpenBao + gradual re-hash on login |
| JWT signing secret | 30 days | OpenBao → graceful rollover (accept old + new for 5 min) |
| Database credentials | 24 hours | OpenBao dynamic secrets (PostgreSQL engine) |
| S3 access keys | 7 days | OpenBao dynamic secrets (AWS engine) |
| Personal access tokens | User-defined expiry, max 1 year | Enforced at creation |

### OpenBao Integration

```go
// pkg/secrets/openbao.go

type SecretManager interface {
    // GetSecret retrieves the current value of a named secret.
    GetSecret(ctx context.Context, path string) (string, error)

    // RotateSecret triggers rotation for secrets that support it.
    RotateSecret(ctx context.Context, path string) error

    // Watch returns a channel that emits when a secret is rotated.
    // Services subscribe to re-load their in-memory copies.
    Watch(ctx context.Context, path string) (<-chan SecretEvent, error)
}
```

### Graceful JWT Rollover

JWT rotation without logging users out:

```go
// internal/auth/jwt_authenticator.go

type JWTAuthenticator struct {
    current  []byte // active signing key
    previous []byte // accepted for 5 minutes after rotation
    rotatedAt time.Time
}

func (a *JWTAuthenticator) Validate(ctx context.Context, token string) (*Claims, error) {
    // Try current key first
    claims, err := validateWithKey(token, a.current)
    if err == nil {
        return claims, nil
    }
    // Fallback: accept previous key during 5-minute grace window
    if time.Since(a.rotatedAt) < 5*time.Minute {
        return validateWithKey(token, a.previous)
    }
    return nil, ErrUnauthorized
}
```

### OpenBao in Docker Compose (dev)

```yaml
# docker-compose.yml addition
openbao:
  image: openbao/openbao:latest
  command: server -dev -dev-root-token-id="dev-root-token"
  environment:
    BAO_DEV_ROOT_TOKEN_ID: dev-root-token
    BAO_LOG_LEVEL: warn
  ports:
    - "8200:8200"
  cap_add:
    - IPC_LOCK
```

In dev mode, OpenBao starts unsealed with a known root token. Production uses auto-unseal
with AWS KMS or an HSM.

---

## New Dependencies

```
golang.org/x/crypto                     — AES-256-GCM, argon2id (already in stack)
github.com/openbao/openbao/api/v2       — OpenBao API client (open source Vault fork, MPL-2.0)
```

No new infrastructure required for dev/test. `EnvKeyRing` and `EnvSecretManager` cover
single-node deployments without OpenBao. Production multi-tenant deployments run OpenBao
for key management, dynamic DB credentials, and secret rotation.

---

## Acceptance Criteria

### Field-Level Encryption
- [ ] `FieldEncryptor.Encrypt` / `Decrypt` implemented with AES-256-GCM
- [ ] `KeyRing` interface with `EnvKeyRing` (dev) and `VaultKeyRing` (prod) implementations
- [ ] Per-tenant key isolation: org_id bound into AAD
- [ ] Key rotation does not require re-encrypting existing data (old keys kept for decryption)
- [ ] PHI nodes are transparently encrypted on write, decrypted on read
- [ ] `content_encrypted = true` nodes return error if key is unavailable (never plaintext fallback)

### Audit Log
- [ ] Hash chain implemented: every entry includes SHA-256 of previous entry
- [ ] `VerifyChain` detects any gap or modification in the log
- [ ] All PHI access events logged with `phi_accessed = true`
- [ ] `audit_log` table has `UPDATE`/`DELETE` revoked for application role
- [ ] Audit middleware attaches to all gRPC and HTTP handlers automatically
- [ ] Audit log retained minimum 7 years (2555-day default retention policy)

### Session Security
- [ ] Idle timeout enforced (configurable, default 15 min for PHI orgs)
- [ ] Absolute timeout enforced (default 8 hours)
- [ ] `last_activity_at` updated on each authenticated request

### Data Residency
- [ ] Right to erasure endpoint implemented
- [ ] `DataErasureJob` deletes all user data including S3 and Valkey
- [ ] Audit log records erasure event (log entry itself never deleted)
- [ ] Retention policies table created with defaults on org creation

### BAA
- [ ] `baa_agreements` table created
- [ ] PHI mode blocked without BAA acceptance
- [ ] BAA acceptance recorded with IP and timestamp

### Compliance Dashboard
- [ ] All 7 admin compliance endpoints implemented
- [ ] Chain verification endpoint returns first broken entry if tampered
- [ ] PHI inventory lists all nodes with `content_encrypted = true`

### Password + Pepper
- [ ] `PasswordHasher.Hash` applies pepper before argon2id hashing
- [ ] `PasswordHasher.Verify` applies pepper before verification
- [ ] Pepper loaded from OpenBao / env at startup — never hardcoded or stored in DB
- [ ] `pepper_version` column on users table supports gradual rotation
- [ ] Unit test: same password + different pepper produces different hash
- [ ] Unit test: verification fails if pepper changes without re-hash

### Secret Auto-Rotation
- [ ] `SecretManager` interface with `EnvSecretManager` (dev) and `OpenBaoSecretManager` (prod)
- [ ] `Watch` channel triggers in-process re-load when OpenBao rotates a secret
- [ ] JWT graceful rollover: old key accepted for 5 minutes after rotation
- [ ] `pepper_version` increments on pepper rotation; re-hash triggered on next login
- [ ] OpenBao added to `docker-compose.yml` (dev mode)
- [ ] `make openbao-setup` Makefile target provisions secrets paths for local dev
- [ ] Rotation schedule documented in `configs/example.env`

### Testing
- [ ] 100% coverage on `pkg/crypto/` (encryption is security-critical)
- [ ] 100% coverage on `pkg/secrets/` (secret management is security-critical)
- [ ] 100% coverage on `internal/audit/` (tamper-evidence logic)
- [ ] Unit test: encrypted content is not readable from raw DB row
- [ ] Unit test: hash chain breaks on any row modification
- [ ] Unit test: key rotation — old ciphertext decryptable with old key, new writes use new key
- [ ] Unit test: JWT rollover — token signed with old key accepted within 5-minute window
- [ ] Unit test: JWT rollover — token signed with old key rejected after 5-minute window
- [ ] Integration test: right to erasure removes all data from DB, Valkey, S3
- [ ] Integration test: BAA gate blocks PHI storage without agreement

---

## Dependencies

- T-004 (PostgreSQL — migrations)
- T-007 (Auth — session management)
- T-098 (Security hardening — shared middleware layer)
- T-101 (Multi-tenancy — org_id, per-tenant keys)
- T-102 (RBAC — permissions, audit log foundation)

## Notes

- Field-level encryption adds ~0.5ms per read for PHI fields (AES-256-GCM is fast).
  Profile before deciding to cache decrypted content (caching PHI in memory has its own risks).
- The audit log hash chain does not prevent deletion — it detects it. For true immutability,
  export logs to an append-only S3 bucket with Object Lock enabled in production.
- SOC2 Type I (point-in-time) is achievable after this feature ships. Type II requires
  6–12 months of evidence that controls operated continuously.
- HIPAA "safe harbor" de-identification requires removing 18 specific identifiers. A
  `Deidentify(node)` function is a `someday` task once the dataset is large enough to test.
