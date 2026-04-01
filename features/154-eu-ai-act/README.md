# T-154 — EU AI Act Compliance (Regulation 2024/1689)

> **Tier: SaaS** — Interface and spec are open source. The AI usage register implementation
> (append-only, certified 3-year retention, REVOKE UPDATE/DELETE enforcement) lives in the
> private `infinitebrain-cloud` module. OSS builds receive a no-op stub. Full compliance
> register requires the managed platform or a commercial license.

## Overview

The EU AI Act is the world's first comprehensive AI regulation. It applies to Infinite Brain
because infinitebrain.io's output is used within the EU (extraterritorial scope, Article 2).

This task establishes:
1. A risk tier assessment of every IB AI feature
2. The AI usage register (machine-readable audit trail of AI operations)
3. GPAI deployer obligations for IB's use of Claude (Anthropic)
4. The technical and legal controls that keep IB in the correct risk tier

The open-source self-hosted IB deployment is largely exempt under Article 2(12).
The hosted cloud (infinitebrain.io) bears the full compliance burden.

---

## Risk Tier Assessment

### Prohibited — IB must never implement

| Prohibited category | Status |
|---|---|
| Subliminal manipulation targeting behavior against user interest | Never build — ADHD nudges must assist users toward their own goals only |
| Social scoring of individuals by public authorities | Not applicable |
| Real-time biometric identification in public spaces | Not applicable |
| AI exploiting vulnerabilities of protected groups | ADHD users are protected — nudges must never coerce |
| Individual employee monitoring for employment decisions | Blocked by T-157 + T-146 |

### High-Risk — IB avoids this tier by design

IB's org intelligence tier (T-138–T-153) is designed to be **NOT** high-risk by enforcing:
- No individual data at the org layer (T-146 anonymization)
- No employment decision support (T-157 prohibition)
- No access to individual-level org metrics (T-149 aggregated-only API)

If any org feature is found to enable individual employee evaluation → it is a bug,
not a feature, and must be removed.

**Healthcare caveat**: AI recommendations on PHI nodes used for clinical decisions would be
high-risk (Annex III §5). IB explicitly prohibits clinical decision use in ToS.
PHI nodes store personal health information for personal reference only.

### Limited Risk — applies to IB; compliance required by Aug 2025

These transparency obligations apply to all AI systems:

| Requirement | IB implementation |
|---|---|
| Bots must identify as AI (Article 52(1)) | T-158 — bots self-identify on first message |
| AI-generated content must be labeled (Article 52(3)) | T-155 — `ai_generated` flag on all AI outputs |
| Users must be able to override AI decisions | T-129 (corrections) + T-156 (explanation) |
| Right to explanation for AI decisions (Article 86) | T-156 — `ai_rationale` on every AI decision |

### Minimal Risk — most IB features

PARA classification, auto-tagging, semantic search, deduplication, daily digest, focus timer,
spaced repetition — all minimal risk. No additional obligations.

---

## GPAI Deployer Obligations (Article 55, effective Aug 2, 2025)

IB uses Claude (Anthropic's GPAI model). Anthropic is the GPAI **provider** (bears model-level
obligations). IB is a GPAI **deployer** — downstream obligations apply:

### IB's deployer obligations

**1. Usage policy compliance**
IB must use Claude only in ways consistent with Anthropic's model card and usage policies.
IB must not instruct Claude to perform prohibited operations (manipulation, social scoring, etc.).

**2. Transparency to users**
Users must be informed that AI is processing their data and making recommendations.
All AI outputs must be labeled (T-155).

**3. AI usage register**
IB must maintain a machine-readable record of AI operations:

```go
// internal/compliance/ai_register.go

type AIUsageRecord struct {
    ID            uuid.UUID
    OrgID         uuid.UUID    // which org
    UserID        *uuid.UUID   // which user (nil for org-level ops)
    Timestamp     time.Time
    Operation     string       // classify, tag, embed, qa, digest, agent_task, ...
    Model         string       // claude-sonnet-4-6, whisper-1, ...
    Provider      string       // anthropic, openai, ...
    InputCategory string       // note_content, voice_transcript, phi_content, ...
    // Note: never log actual content — only category
    TokensIn      int
    TokensOut     int
    Safeguards    []string     // ["phi_sanitized", "prompt_guard_applied", "org_anonymized"]
    RiskTier      string       // minimal, limited, high — determined per operation
    CreatedAt     time.Time
}
```

This register is:
- Append-only (never UPDATE or DELETE)
- Queryable for audits: "show all AI operations on PHI data in Q1 2026"
- Retained for 3 years (audit requirement)
- Exportable for regulatory inspection

**4. Human oversight mechanisms**
For any AI decision that affects a user's data or recommendations, a human must be able to
review and override it. This is satisfied by T-129 (corrections) and T-156 (explanation).

---

## Open-Source Exemption (Article 2(12))

Self-hosted AGPL-3.0 IB is exempt from high-risk obligations **if**:
- The system is not high-risk (our tier assessment confirms this)
- The self-hoster is not offering a commercial service

**Self-hoster becomes provider** when:
- They deploy IB for their employees (commercial org use)
- They use IB to make employment decisions
- In this case: they bear the deployer obligations, not Anthropic or IB maintainers

IB documentation must clearly state:
- "If you deploy IB for your organization and use it in employment-related decisions, you
  become a provider under the EU AI Act and bear applicable compliance obligations."

---

## SECURITY.md and Legal Documents

The following must be written or updated:

**`docs/SECURITY.md`** — add EU AI Act section:
- Risk tier assessment summary
- How to report prohibited use violations
- Self-hoster obligations

**`docs/COMPLIANCE.md`** — comprehensive:
- SOC2 + HIPAA (T-104)
- EU AI Act (this task)
- GDPR (overlaps: right to erasure, data minimization already implemented)
- Open-source exemption scope

**`docs/AI-USAGE-POLICY.md`** — GPAI deployer documentation:
- Which models IB uses and for what
- What data categories are sent to AI providers
- What safeguards are applied before every AI call
- How to configure self-hosted IB to use alternative providers

---

## Acceptance Criteria

- [ ] Risk tier assessment documented for every AI feature in IB
- [ ] `AIUsageRecord` entity + append-only table (`ai_usage_register`)
- [ ] Every AI operation writes a usage record (model, operation, safeguards, risk tier)
- [ ] Register queryable for audits; export endpoint for regulatory inspection
- [ ] Register retained for 3 years; REVOKE UPDATE, DELETE from app role
- [ ] `docs/COMPLIANCE.md` covering SOC2, HIPAA, EU AI Act, GDPR
- [ ] `docs/AI-USAGE-POLICY.md` covering GPAI deployer obligations
- [ ] Self-hoster obligation notice in README + docs
- [ ] 90% test coverage on compliance package

## Dependencies

- T-104 (SOC2/HIPAA — audit log infrastructure; same append-only pattern)
- T-155 (AI transparency labeling — implements the Limited Risk disclosure requirements)
- T-156 (Right to explanation — implements Article 86)
- T-157 (Employment prohibition — keeps org tier out of high-risk)
- T-158 (Bot disclosure — implements Article 52(1))
- T-120 (Event sourcing — AI usage register is append-only event stream)
