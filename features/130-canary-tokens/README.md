# T-130 — Canary Tokens in Honeypot

## Overview

The fake credentials in the honeypot `.env` (T-099) are registered as canary tokens.
When an attacker takes those credentials and tries to use them — even from a completely
different network — we get an immediate alert with their IP, timestamp, and tool signature.

This detects breaches that the honeypot endpoints themselves don't catch: an attacker who
exfiltrates the file but doesn't probe the API.

## What Are Canary Tokens

A canary token is a credential that looks real but triggers a callback when used.
canarytokens.org provides free tokens for AWS keys, API keys, and generic webhooks.
When the token is used anywhere in the world, canarytokens.org sends an HTTP callback.

## Honeypot Credential Types

```bash
# configs/honeypot.env (checked into repo — this IS the bait)

# These look like real credentials but are registered canary tokens
ANTHROPIC_API_KEY=sk-ant-canary-xxxxxxxxxxxxxxxxxxxx     # registered at canarytokens.org
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7CANARY00                  # AWS canary key
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/CANARY000
DATABASE_URL=postgres://admin:C4n4ry_P4ssw0rd@db.internal:5432/prod
JWT_SECRET=ThisIsAFakeJWTSecretThatLooksRealEnough1234
```

## Alert Flow

```
Attacker uses canary credential
    │
    └── canarytokens.org → POST /api/v1/security/canary-alert
            │
            ├── Parse: IP, timestamp, user-agent, credential type
            ├── Create security_incident (T-104)
            ├── Severity = 'critical' (credential outside the system = active breach)
            ├── Alert via: Slack webhook + email + PagerDuty (if configured)
            └── Log to immutable audit log (T-104)
```

## Callback Endpoint

```go
// internal/security/canary.go

// POST /api/v1/security/canary-alert — public, no auth required
// (canarytokens.org calls this)
func (h *CanaryHandler) Alert(ctx context.Context, req *CanaryAlertRequest) error {
    incident := SecurityIncident{
        OrgID:       uuid.Nil,  // unknown — system-level incident
        Severity:    "critical",
        Type:        "canary_token_triggered",
        PHIInvolved: false,
        Description: fmt.Sprintf("Canary token '%s' used from IP %s at %s",
            req.TokenName, req.IP, req.Timestamp),
    }
    // Create incident, alert, audit log
}
```

## Self-Hosted Option

For production self-hosters who don't want canarytokens.org in the loop:
generate fake AWS credentials using the `canarytokens` open source library and
host the callback endpoint internally.

## Acceptance Criteria

- [ ] `configs/honeypot.env` with realistic-looking fake credentials checked into repo
- [ ] Each credential type documented as a canary token in comments
- [ ] `POST /api/v1/security/canary-alert` endpoint (public, HMAC-verified from canarytokens.org)
- [ ] Alert creates `security_incidents` record with severity=critical
- [ ] Alert triggers notification (Slack/email) with IP and token type
- [ ] Alert recorded in immutable audit log
- [ ] Self-hosted canary generation documented in SECURITY.md
- [ ] 90% test coverage on alert handler

## Dependencies

- T-099 (Honeypot endpoints — the bait is the honeypot.env)
- T-104 (Compliance — security_incidents table + audit log)
