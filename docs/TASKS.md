# Infinite Brain — Task Registry

> **AI INSTRUCTION**: Read this file at the start of every session. Update task status before
> and after working on any task. Never mark a task complete unless all acceptance criteria are met
> and all tests pass.
>
> **PROJECT STATE**: Ground zero — planning phase only. No implementation has started beyond
> the initial scaffold (T-001, T-002, T-003, T-090, T-091). All specs and architecture docs
> are the plan; nothing is locked. Any spec, task, or architectural decision can be revised
> before implementation begins.

---

## Status Legend

| Status | Meaning |
|---|---|
| `planned` | Defined, not yet started |
| `in_progress` | Actively being implemented |
| `blocked` | Waiting on a dependency |
| `someday` | Post-MVP — parked intentionally, not in current scope |
| `canceled` | Will not be implemented (reason in Notes) |
| `completed` | All criteria met, tests pass, merged |

---

## Build Order Rationale

Tasks are ordered by the sequence in which they must be built. Each phase depends on the
previous. Security and compliance come before features — not at the end. This signals
engineering discipline and is what makes this a portfolio-grade project.

```
Phase 0:  Repository foundation   → what someone sees when they clone
Phase 1:  Infrastructure           → database, cache, secrets
Phase 2:  Server layer             → connect-go, health, observability
Phase 3:  Security                 → before any business feature
Phase 4:  Identity & Access        → auth, users, SSO, RBAC, tenancy
Phase 5:  Resilience patterns      → circuit breaker, idempotency, events
Phase 6:  Data layer               → event sourcing (foundational)
Phase 7:  Capture engine           → first business feature
Phase 8:  AI engine                → intelligence layer + cost + prompt versioning
Phase 9:  Knowledge graph          → the brain's long-term memory
Phase 10: ADHD / Tasks             → workflow engine
Phase 11: External interfaces      → MCP, bots, integrations
Phase 12: Intelligence features    → digests, reviews, insights
Phase 13: Production readiness     → k8s, feature flags, load tests
Tier 2:   Advanced AI              → consensus, dedup, preferences
Tier 3:   Cutting edge             → SBOM, mTLS, canary tokens
Someday:  Post-MVP                 → clearly parked
```

---

## Phase 0 — Repository Foundation

> First impression. Anyone cloning the repo evaluates this before reading a line of code.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-001 | Project setup: Go module, folder structure, Makefile | `completed` | features/001-project-setup/ | Initial scaffold |
| T-105 | LICENSE (AGPL-3.0) | `planned` | — | Actual LICENSE file in repo root |
| T-106 | README.md — portfolio-grade | `planned` | — | Badges, Mermaid architecture diagram, quick start, stack rationale, links to ADRs |
| T-107 | Open source hygiene | `planned` | — | CONTRIBUTING.md, SECURITY.md, CHANGELOG.md, .github/ISSUE_TEMPLATE/, .github/pull_request_template.md |
| T-108 | Dockerfile — multi-stage + distroless | `planned` | features/108-dockerfile/ | Builder stage (Go 1.26.1) + gcr.io/distroless/static final; non-root user; no shell |
| T-109 | Architecture Decision Records (ADRs) | `planned` | docs/decisions/ | One ADR per key decision: connect-go, Valkey, River, OpenBao, Atlas, pgvector, AGPL-3.0 |

---

## Phase 1 — Infrastructure Foundation

> Must be solid before anything runs on top of it.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-002 | Core configuration system (cleanenv, env + yaml) | `completed` | features/002-config/ | pkg/config — 7 tests |
| T-003 | Structured logger (slog) | `completed` | features/003-logger/ | pkg/logger — 4 tests |
| T-004 | PostgreSQL 18.3 + Atlas migrations | `completed` | features/004-database/ | pgvector enabled; Atlas schema-as-code; reversible migrations |
| T-005 | Valkey 9 connection pool | `completed` | features/005-valkey/ | Replaces Redis; pkg/cache; connection health check |
| T-090 | Docker Compose dev environment | `completed` | features/090-docker/ | PostgreSQL 18, Valkey 9, MinIO, OpenBao, Jaeger |

---

## Phase 2 — Server Layer

> The HTTP/gRPC surface. Observability wired in from the start — not added later.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-112 | connect-go server (REST + gRPC + gRPC-Web, graceful shutdown) | `completed` | features/112-server/ | Single handler serves all three protocols; replaces Chi + separate gRPC server; buf + protovalidate |
| T-103 | Protocol Buffers — service definitions + buf codegen | `completed` | features/103-grpc/ | api/proto/; common/v1, capture/v1, ai/v1, knowledge/v1, memory/v1; make proto target |
| T-110 | Health + readiness endpoints | `completed` | features/110-health/ | GET /health/live (liveness), GET /health/ready (readiness: DB + Valkey + migrations); used by k8s probes |
| T-111 | Observability foundation | `planned` | features/111-observability/ | Correlation/trace IDs on every request; slog fields; OTEL traces to Tempo; Prometheus /metrics; request ID in response headers |
| T-091 | CI pipeline (GitHub Actions) | `completed` | features/091-ci/ | test + lint + build + security scan; Go 1.26.1, pgvector/pgvector:pg18, valkey:9-alpine |

---

## Phase 3 — Security

> Security is built before features, not added after. This ordering is intentional and visible.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-098 | Security hardening middleware | `planned` | features/098-security/ | HTTP security headers; Valkey sliding-window rate limiter; account lockout; prompt injection guard (PromptGuard) |
| T-099 | Honeypot endpoints | `planned` | features/099-honeypot/ | 8 fake endpoints; hit logging; progressive auto-block (2→24h, 5→7d, 10→permanent); fake .env with realistic credentials |
| T-104 | SOC2 + HIPAA compliance | `planned` | features/104-compliance/ | Field-level AES-256-GCM; OpenBao key management; tamper-evident audit log (hash chain); salt+pepper passwords; auto-rotation; BAA support; right to erasure |

---

## Phase 4 — Identity & Access

> Who can do what. Multi-tenancy from day one — retrofitting it later is the mistake that breaks everything.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-007 | Auth system — JWT + argon2id + pepper + refresh token rotation | `planned` | features/007-auth/ | JWTAuthenticator implements Authenticator interface; stateless; argon2id with pepper from OpenBao |
| T-008 | User model + registration / login API | `planned` | features/008-users/ | internal/auth; users table; email verification; password reset |
| T-101 | Multi-tenancy — organizations + members | `planned` | features/101-multi-tenancy/ | organizations + org_members tables; org_id on all data tables; PostgreSQL RLS; personal org auto-created on signup |
| T-100 | Zitadel SSO — OIDC integration | `planned` | features/100-zitadel-sso/ | OIDCAuthenticator via go-oidc/v3; same Authenticator interface as T-007; personal access tokens (ibpat_ prefix); Zitadel in docker-compose |
| T-102 | RBAC — roles and permissions | `planned` | features/102-rbac/ | owner / admin / editor / viewer; Can(role, permission); connect-go interceptor; org_invites; append-only audit_log; Zitadel role sync |

---

## Phase 5 — Resilience Patterns

> Architecture patterns that protect every feature built after this. Retrofitting these is expensive.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-113 | Resilience — circuit breaker + retry + timeout | `planned` | features/113-resilience/ | pkg/resilience; circuit breaker for all external calls (AI, Whisper, webhooks); exponential backoff with jitter; configurable timeouts per call type |
| T-114 | Idempotency keys | `planned` | features/114-idempotency/ | Idempotency-Key header on all mutating endpoints; Valkey-backed deduplication with TTL; prevents duplicate captures from bot retries |
| T-115 | Domain events — internal pub/sub | `planned` | features/115-domain-events/ | NoteCreated, NodeLinked, TaskCompleted, ChunkStarted etc.; River-backed subscribers; loose coupling between domains; no direct cross-domain calls |

---

## Phase 6 — Data Layer (Event Sourcing)

> The foundational persistence pattern. Must be built before any business aggregate.
> The knowledge graph, tasks, captures, and AI memory are all projections of the event log.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-120 | Event sourcing — EventStore, aggregates, projectors | `planned` | features/120-event-sourcing/ | domain_events table (append-only); Aggregate base; sync + async projectors; ProjectionRebuilder; temporal queries (LoadAt); event upcasting for schema evolution |

---

## Phase 7 — Capture Engine

> First business feature. Every capture path must be zero-friction.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-010 | Note model + CRUD API | `planned` | features/010-notes/ | Core capture unit; org-scoped; NodeAggregate (T-120); full test coverage |
| T-011 | Voice note upload + transcription (Whisper) | `planned` | features/011-voice-notes/ | S3 upload; River job for async transcription; result back-fills node content |
| T-013 | Email capture (inbound webhook → note) | `planned` | features/013-email-capture/ | Postal / Mailgun inbound; parses headers + body; lands in inbox |
| T-014 | Webhook capture endpoint (generic) | `planned` | features/014-webhooks/ | Generic POST /capture; any bot or integration sends here; HMAC signature validation |
| T-015 | Inbox queue — unprocessed captures | `planned` | features/015-inbox/ | Staging area before AI processing; River job per item; visible to user as "processing" |
| T-128 | Semantic de-duplication on capture | `planned` | features/128-semantic-dedup/ | Cosine similarity check before node creation (threshold 0.92); surface: link/merge/create_anyway |
| T-012 | File attachment support (S3-compatible) | `someday` | features/012-attachments/ | Post-MVP |

---

## Phase 8 — AI Engine

> The intelligence layer. Provider interface keeps every feature AI-provider-agnostic.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-020 | AI provider abstraction | `planned` | features/020-ai-provider/ | Provider interface: Complete, Embed, Transcribe; ClaudeProvider + OpenAIProvider; factory; ProviderChain fallback |
| T-016 | AI session memory store | `planned` | features/016-ai-session-memory/ | agent_memories table; embedding index; ContextLoader; shared memory bus for parallel agents |
| T-021 | Auto-classification — PARA category routing | `planned` | features/021-ai-classify/ | Structured JSON output; versioned prompts in internal/ai/prompts/ |
| T-022 | Auto-tagging — entities, topics, projects | `planned` | features/022-ai-tagging/ | Runs after classification; tags stored on node |
| T-023 | Semantic search (pgvector embeddings) | `planned` | features/023-semantic-search/ | text-embedding-3-small default; HNSW index; cosine similarity; cursor pagination |
| T-121 | Hybrid search — BM25 + vector (RRF) | `planned` | features/121-hybrid-search/ | Reciprocal Rank Fusion; ts_rank + cosine; GIN index; mode=hybrid/bm25/vector selectable |
| T-024 | Question answering over personal knowledge base | `planned` | features/024-ai-qa/ | RAG via hybrid search (T-121) → inject as context → streaming answer via connect-go |
| T-122 | Prompt versioning + A/B testing | `planned` | features/122-prompt-versioning/ | Prompts as versioned code; traffic splitting; user correction = ground truth signal; auto-graduation |
| T-124 | AI cost attribution | `planned` | features/124-ai-cost-attribution/ | Per-call token + cost tracking; daily aggregates; threshold alerts; billing API; metered billing foundation |
| T-123 | Memory compression (nightly consolidation) | `planned` | features/123-memory-compression/ | Cluster + summarize old agent_memories; expires originals; River cron at 02:00 |
| T-097 | MCP provider adapter — plug any MCP-compatible AI model | `planned` | features/097-mcp-provider/ | MCPProvider implements Provider interface; stdio + HTTP transport; capability discovery; ProviderChain fallback |

---

## Phase 9 — Knowledge Graph

> The brain's long-term memory. Every entity is a node. Every relationship is a typed edge.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-028 | Knowledge graph — nodes + edges | `planned` | features/028-knowledge-graph/ | Universal nodes + edges tables; node types: project/note/task/event/media/rule/decision/insight/contact; relation types: implements/solves/contradicts/relates/inspired_by/blocks/part_of |
| T-036 | Relevance decay + review ladder | `planned` | features/036-relevance-decay/ | 6-stage FSM; silence = resurface; Yes = biannual; 4× No over 16 months = hard delete |
| T-029 | Cross-project insight linker (nightly cron) | `planned` | features/029-insight-linker/ | pgvector cosine similarity (threshold 0.82); AI validation; creates insight node + 2 edges; deduplication; surfaces in digest |

---

## Phase 10 — Tasks & ADHD Engine

> The workflow layer. ADHD-specific design is the product differentiator.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-030 | Task model + CRUD API | `planned` | features/030-tasks/ | Linked to knowledge graph nodes; priority; deadline; River job for AI priority scoring |
| T-034 | AI-assisted priority scoring | `planned` | features/034-priority/ | Scores tasks by urgency + energy required + context; updates on capture or schedule change |
| T-035 | Now / Next / Later triage view | `planned` | features/035-triage/ | Derived from priority scores; user can override; API endpoint returns 3 buckets |
| T-042 | Distraction capture (log without breaking focus) | `planned` | features/042-distraction-capture/ | Single-field quick-capture endpoint; lands in inbox; does not interrupt current chunk |
| T-046 | "Where was I?" — context restore on task resume | `planned` | features/046-where-was-i/ | Loads recent notes, last AI session memory, open edges for the task; returns summary |
| T-048 | Daily chunk planner | `planned` | features/048-chunk-planner/ | N chunks/day (default 16); type-mixed (work/chore/exercise/personal/free); order-free; AI task suggestion at chunk-start; River timer per chunk |
| T-031 | Project model + CRUD | `someday` | features/031-projects/ | Post-MVP |
| T-032 | Sub-tasks and dependencies | `someday` | features/032-subtasks/ | Post-MVP |
| T-033 | Time tracking | `someday` | features/033-time-tracking/ | Post-MVP |
| T-041 | Hyperfocus guard | `someday` | features/041-hyperfocus-guard/ | Post-MVP |
| T-043 | Energy-aware scheduling | `someday` | features/043-energy-schedule/ | Post-MVP |

---

## Phase 11 — External Interfaces

> How users and AI agents talk to Infinite Brain.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-096 | MCP server — expose Infinite Brain as MCP tools + resources | `planned` | features/096-mcp-server/ | 15 tools + 4 resources; stdio (Claude Code) + HTTP/SSE; ibpat_ auth; separate cmd/mcp binary |
| T-070 | Telegram bot | `planned` | features/070-telegram/ | Primary user interface; capture, query, triage commands |
| T-071 | WhatsApp integration | `planned` | features/071-whatsapp/ | Twilio / Meta Cloud API; voice notes + text capture |
| T-072 | Slack integration | `planned` | features/072-slack/ | Slash commands + message save; team-aware (uses org from workspace) |
| T-076 | Obsidian integration | `planned` | features/076-obsidian/ | Local REST API plugin + CLI sync; bidirectional |
| T-077 | Apple Notes integration | `planned` | features/077-apple-notes/ | AppleScript + Shortcuts; capture from macOS/iOS |
| T-073 | GitHub integration | `someday` | features/073-github/ | Post-MVP |
| T-074 | Readwise integration | `someday` | features/074-readwise/ | Post-MVP |
| T-075 | Web clipper | `someday` | features/075-web-clipper/ | Post-MVP |
| T-044 | Apple Watch notifications | `someday` | features/044-watch-notifications/ | Post-MVP |
| T-045 | Haptic task-switch reminders | `someday` | features/045-task-switch/ | Post-MVP |

---

## Phase 12 — Intelligence Features

> Proactive AI that works while you sleep.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-025 | Daily digest generation | `planned` | features/025-daily-digest/ | River cron at 07:00; includes: due tasks, new insights from T-029, review prompts from T-036, chunk plan for the day |
| T-026 | Weekly review generation | `planned` | features/026-weekly-review/ | River cron Sunday evening; patterns, accomplishments, open loops, energy analysis |
| T-129 | Personal AI preferences model | `planned` | features/129-ai-preferences/ | User correction signals train per-user preference profile; injected into classify/tag prompts after 50+ ops; fully user-owned + deletable |

---

## Phase 13 — Production Readiness

> What separates "runs on my machine" from "runs in production".

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-093 | Observability — full stack (Tempo, Loki, Grafana) | `planned` | features/093-observability/ | Builds on T-111 foundation; dashboards for: capture rate, AI latency, error rate, active sessions, PHI access |
| T-116 | Feature flags | `planned` | features/116-feature-flags/ | DB-backed; OpenFeature-compatible interface; flags.IsEnabled(ctx, "new-classifier", orgID); used for gradual rollouts and A/B of AI prompts |
| T-117 | Kubernetes + Helm chart | `planned` | features/117-kubernetes/ | deploy/helm/; Deployment + Service + HPA + PDB + ConfigMap; readiness/liveness probes wired to T-110; resource limits defined |
| T-118 | Load testing (k6) | `planned` | features/118-load-tests/ | tests/load/; scripts for: capture burst (100 concurrent), semantic search under load, AI pipeline throughput; SLO targets documented |
| T-119 | API versioning + deprecation policy | `planned` | features/119-api-versioning/ | Policy document: what triggers v2, grace period for v1, deprecation headers (Sunset, Deprecation), changelog requirement |

---

## Tier 2 — Advanced AI (after core loop proven)

> Builds on the AI engine. Requires data from real usage to be meaningful.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-127 | Multi-model consensus classifier | `planned` | features/127-multi-model-consensus/ | 2 models run concurrently; majority vote; disagreement → flagged for user review; always used for PHI nodes |

---

## Tier 3 — Security Excellence

> Supply chain and network security. Shows depth beyond application-layer security.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-125 | SBOM + signed releases (cosign / sigstore) | `planned` | features/125-sbom-signed-releases/ | Syft generates SBOM on every release; cosign keyless signing; Docker images pinned to SHA digests |
| T-126 | mTLS internal gRPC communication | `planned` | features/126-mtls/ | Services present client certificates; OpenBao PKI issues/rotates certs in prod; TLS 1.3 minimum |
| T-130 | Canary tokens in honeypot credentials | `planned` | features/130-canary-tokens/ | Fake credentials registered at canarytokens.org; use outside the system triggers critical incident |

---

---

## Tier 4 — Project Intelligence Platform

> IB as an intelligent domain database: business rules, requirement-test triad, agent orchestration.
> This tier transforms IB from a second brain into an autonomous development intelligence layer.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-131 | `BusinessRule` node type + conflict detection | `planned` | features/131-business-rules/ | Rules are versioned via event sourcing; semantic conflict detection between rules; rules injected into agent task context |
| T-132 | `Requirement` → `AcceptanceCriteria` structured linking | `planned` | features/132-requirements/ | Requirement nodes link to machine-verifiable acceptance criteria; criteria drive test generation |
| T-133 | Test generation from acceptance criteria | `planned` | features/133-test-generation/ | AI generates Go test stubs from acceptance criteria; tests are stored as nodes; passing = task proven done |
| T-134 | `AgentTask` entity + agent dispatch loop | `planned` | features/134-agent-tasks/ | IB decomposes goals into AgentTask nodes; dispatches agents with full KB context; verifies via acceptance tests |
| T-135 | Context API — external AI tools query IB | `planned` | features/135-context-api/ | `GET /api/v1/context?query=...` returns relevant rules + ADRs + requirements; Cursor/Copilot/Claude Code integration |
| T-136 | Gap analysis — uncovered requirements detection | `planned` | features/136-gap-analysis/ | IB scans codebase + KB; detects requirements with no corresponding implementation or tests |
| T-137 | Architecture drift detection | `planned` | features/137-arch-drift/ | IB compares ADRs and documented patterns against actual code; alerts when code diverges from documented intent |

---

## Tier 5 — Organizational Intelligence

> IB as the operating system for entire organizations. All employees contribute; IB synthesizes
> across silos. Identifies what moves the needle, surfaces what the org doesn't know it doesn't
> know, and preserves institutional knowledge beyond individual tenure.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-138 | Relationship + interaction graph | `planned` | features/138-relationship-graph/ | People nodes + Interaction edges; auto-extract from meeting notes, emails, calls; "prepare me for my meeting with X" |
| T-139 | Decision journal + pattern detection | `planned` | features/139-decision-journal/ | Decisions captured with context + rationale + outcome; IB resurfaces similar past decisions; identifies personal/team decision patterns |
| T-140 | Expertise graph — infer who knows what | `planned` | features/140-expertise-graph/ | Derived from captured knowledge, not org charts; "who has dealt with Stripe webhooks?" returns ranked people nodes |
| T-141 | Cross-user knowledge synthesis | `planned` | features/141-org-synthesis/ | Surfaces connections between what different users capture; team-level insight from individual knowledge contributions |
| T-142 | Needle-mover analysis | `planned` | features/142-needle-movers/ | Correlates internal activity signals to business outcomes; identifies real leading indicators; cross-silo pattern detection |
| T-143 | Knowledge concentration risk | `planned` | features/143-knowledge-risk/ | Detects single-points-of-failure in org knowledge; generates handoff documents for departing employees; "if X left today..." |
| T-144 | Organizational health dashboard | `planned` | features/144-org-health/ | Decision velocity, knowledge flow rate, meeting ROI, context loss rate, dependency concentration — org-level metrics |

---

## Tier 5b — Safe Contribution Infrastructure

> The technical guarantees that make the org brain a safe space to contribute.
> Without this, nobody shares. Without sharing, the org brain is empty.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-145 | Knowledge visibility scopes — personal / team / org | `planned` | features/145-visibility-scopes/ | `visibility` field on every node; explicit publish action (never auto-escalate); each scope has a different privacy contract |
| T-146 | Org-layer anonymization — strip attribution at ingestion | `planned` | features/146-org-anonymization/ | Name/ID stripped when node published to org; k-anonymity enforcement (n≥5) on all org queries; individual queries rejected at service layer |
| T-147 | Idea pipeline — anonymous submission → cluster → surface | `planned` | features/147-idea-pipeline/ | Semantic dedup of ideas; cluster weight grows with independent submissions; scored against business gaps; surfaced without attribution |
| T-148 | Differential privacy for org metrics | `planned` | features/148-differential-privacy/ | Calibrated noise on org-level counts; week-bucket timestamps; prevents timing-based re-identification |
| T-149 | Org insights API — aggregated only | `planned` | features/149-org-insights-api/ | Org queries return patterns only; group size enforced; no endpoint for individual contribution data; consent-based attribution |

---

## Tier 5c — True North: Alignment Engine

> Every task, project, and agent action evaluated against a declared honest objective.
> The system that tracks *whether* you're moving in the right direction, not just *how fast*.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-150 | `TrueNorth` node type — honest objectives + values + constraints | `planned` | features/150-true-north/ | Root node of the knowledge graph; declared honest objectives (not just mission statements); values, real constraints, anti-patterns |
| T-151 | Alignment scoring — score any task/project against True North | `planned` | features/151-alignment-scoring/ | 0.0–1.0 alignment score per node; explains why; injected into task prioritization and agent dispatch |
| T-152 | Drift detection — gap between declared True North and actual decisions | `planned` | features/152-drift-detection/ | Analyzes decision history vs. stated objectives; surfaces when the org is drifting from its True North |
| T-153 | True North-driven task prioritization | `planned` | features/153-true-north-priority/ | Daily task list ordered by alignment score × impact × urgency; replaces pure urgency/importance matrix |

---

## Tier 5d — EU AI Act Compliance

> Regulation 2024/1689 effective Aug 2024. Extraterritorial — applies to infinitebrain.io hosted SaaS.
> Open-source self-hosted IB is largely exempt. Hosted cloud is not. Aug 2025: GPAI deployer rules.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-154 | EU AI Act documentation — AI usage register + risk assessment | `planned` | features/154-eu-ai-act/ | Machine-readable AI usage register (which model, which task, which safeguards); risk tier assessment per feature; deployer obligations for Claude (GPAI) |
| T-155 | AI transparency labeling — all AI outputs marked | `planned` | features/155-ai-transparency/ | `ai_generated`, `ai_model`, `ai_confidence` on every AI-generated node/response; surfaced in API and UI |
| T-156 | Right to explanation — "why did IB do this?" | `planned` | features/156-ai-explainability/ | Every AI decision (classify, prioritize, recommend) returns human-readable explanation; stored as `ai_rationale` on node |
| T-157 | Employment use prohibition — technical + legal controls | `planned` | features/157-employment-prohibition/ | ToS clause prohibiting HR/employment use of org tier; API blocks returning individual data in org context; org admin attestation on setup |
| T-158 | Bot AI identity disclosure — all bots self-identify | `planned` | features/158-bot-disclosure/ | Telegram/WhatsApp/Slack bots identify as AI on first message + in bio; required under EU AI Act Limited Risk tier |

---

## Tier 5e — Recursive Org Architecture

> Extends the flat `org_id` model to a full organizational unit hierarchy.
> Same knowledge graph, scoped at arbitrary depth: individual → team → squad → unit → org.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-159 | `OrgUnit` hierarchy — teams, squads, units as first-class entities | `planned` | features/159-org-units/ | `org_units` table with `parent_unit_id` self-reference; `unit_id` on all nodes; visibility scoped to unit level; same engine, arbitrary depth |
| T-160 | Unit-scoped knowledge graph views | `planned` | features/160-unit-brain/ | Each unit has its own "brain view" — nodes visible at that unit level; brain at each level = filtered graph; upward propagation with anonymization |
| T-161 | Cross-unit insight linking | `planned` | features/161-cross-unit-insights/ | IB detects when knowledge in one unit is relevant to another; surfaces connections without revealing individual attribution |
| T-164 | `CoherenceScore` — measure and track balance per unit | `planned` | features/164-coherence-score/ | 6-component score: knowledge density, connection density, True North alignment, decision coverage, freshness, concentration risk; computed per unit on schedule |
| T-165 | Coherence dashboard + entropy alerts | `planned` | features/165-coherence-dashboard/ | Per-unit coherence breakdown; "what to fix" action items; alert when score drops below threshold |
| T-166 | Open unit hierarchy — user-defined level names, infinite depth | `planned` | features/166-open-hierarchy/ | Remove `unit_type` enum; any name valid; depth computed via recursive CTE; hierarchy mirrors any real-world structure |
| T-167 | Knowledge freshness decay + stale node detection | `planned` | features/167-knowledge-freshness/ | Nodes have a freshness score that decays over time; stale nodes surfaced for review; freshness feeds coherence score |
| T-168 | Iteration retrospective — structured capture after each build cycle | `planned` | features/168-retrospective/ | Captures: retry count, plan vs. actual delta, test quality signal, True North alignment score, human corrections; stored as structured event |
| T-169 | Pattern detection across retrospectives | `planned` | features/169-pattern-detection/ | Identifies recurring failure types across iterations; threshold-based trigger for methodology update candidates |
| T-170 | Methodology evolution engine — versioned updates to prompts/templates/rules | `planned` | features/170-methodology-evolution/ | Every methodology artifact (decomposition prompt, test template, context retrieval) is versioned; updates traced to retrospectives; outcomes measured |
| T-171 | Alignment guard on learning — reject updates that reduce True North alignment | `planned` | features/171-alignment-guard/ | Methodology updates that improve efficiency but reduce alignment are blocked; dual optimization: efficiency AND direction |

---

## Tier 5f — Trusted Ingestion + Prompt Injection Defense

> Every external data source is attacker-controlled content that flows through AI processing.
> This tier is the security architecture for untrusted input at scale.

| ID | Task | Status | Feature Spec | Notes |
|---|---|---|---|---|
| T-177 | PromptGuard — prompt injection detection + sanitization | `planned` | features/177-prompt-guard/ | Content-instruction boundary in all AI prompts; output schema validation; pattern catalog; trust levels per source; canary phrase system; 100% coverage |
| T-178 | AI behavioral anomaly detection — catch successful injections | `planned` | features/178-injection-detection/ | Statistical baseline per operation; canary leak detection; schema violation tracking; source reputation + auto-quarantine |
| T-172 | PM connector interface + webhook framework | `planned` | features/172-pm-connectors/ | `PMConnector` interface; HMAC webhook verification; all imported content through PromptGuard; external_id preserved |
| T-173 | Jira integration | `planned` | features/172-pm-connectors/ | OAuth 2.0; JQL filter import; bidirectional status sync; webhook handler |
| T-174 | Asana integration | `planned` | features/172-pm-connectors/ | OAuth 2.0; task search import; webhook handler; status mapping |
| T-175 | GitHub Projects integration (extends T-073) | `planned` | features/172-pm-connectors/ | Projects V2 GraphQL; GitHub App auth; webhook HMAC-SHA256 |
| T-176 | Linear integration | `planned` | features/172-pm-connectors/ | OAuth 2.0; GraphQL import by team/cycle; webhook verification |

---

## The Meta-Loop — IB Builds IB

> This is not a task tier — it is the development methodology.
> All specs, decisions, and architecture in this project feed IB's own knowledge graph.
> IB tracks its own construction using the same features it is being built to provide.

| ID | Task | Status | Notes |
|---|---|---|---|
| T-162 | Import all feature specs into IB knowledge graph | `planned` | Every `features/*/README.md` becomes a node; acceptance criteria become linked `Requirement` nodes (T-132) |
| T-163 | Self-hosted IB dev instance for project management | `planned` | Run IB locally; use it to manage IB's own development; close the meta-loop |

---

## Someday / Post-MVP

> Well-defined but not in scope until core loop is proven.

| ID | Task | Notes |
|---|---|---|
| T-012 | File attachments (S3) | Post-MVP |
| T-027 | Context restoration (standalone) | Covered by T-046 at MVP |
| T-031 | Projects CRUD | Post-MVP |
| T-032 | Sub-tasks + dependencies | Post-MVP |
| T-033 | Time tracking | Post-MVP |
| T-040 | Focus timer (Pomodoro+) | Superseded by T-048 |
| T-041 | Hyperfocus guard | Post-MVP |
| T-043 | Energy-aware scheduling | Post-MVP |
| T-044 | Apple Watch notifications | Post-MVP |
| T-045 | Haptic task-switch reminders | Post-MVP |
| T-047 | AI daily planning assistant | Superseded by T-048 |
| T-050 | Calendar events | Post-MVP |
| T-051 | Google Calendar sync | Post-MVP |
| T-052 | Apple Calendar sync | Post-MVP |
| T-053 | Unified timeline | Post-MVP |
| T-054 | AI meeting scheduling | Post-MVP |
| T-060 | Contacts / CRM | Post-MVP |
| T-061 | Interaction history | Post-MVP |
| T-062 | Org relationship graph | Post-MVP |
| T-063 | Follow-up reminders | Post-MVP |
| T-073 | GitHub integration | Post-MVP |
| T-074 | Readwise integration | Post-MVP |
| T-075 | Web clipper | Post-MVP |
| T-078 | Notion integration | Post-MVP |
| T-079 | Evernote integration | Post-MVP |
| T-080 | Full-text search | Covered by T-023 at MVP |
| T-081 | Semantic search (standalone) | Covered by T-023 at MVP |
| T-082 | Unified search API | Post-MVP |
| T-083 | OneNote integration | Post-MVP |
| T-084 | Joplin integration | Post-MVP |
| T-085 | BetterNotes integration | Post-MVP |

---

## Canceled

| ID | Task | Reason |
|---|---|---|
| T-006 | HTTP server (Chi router) | Replaced by connect-go (T-112) — single handler serves REST + gRPC + gRPC-Web |
| T-092 | Database migration strategy (goose) | Covered by Atlas in T-004 — goose replaced with Atlas |
| T-094 | Rate limiting task (standalone) | Absorbed into T-098 (security hardening) |
| T-095 | API documentation (OpenAPI/Swagger) | connect-go + buf generates OpenAPI 3.1 from proto files automatically |

---

## Completed

| ID | Task | Completed | Notes |
|---|---|---|---|
| T-112 | connect-go Ping service handler | 2026-04-01 | internal/ping; unary handler; 2 integration tests |
| T-110 | Health + readiness endpoints | 2026-04-01 | internal/health; GET /health/live, GET /health/ready |
| T-103 | Protocol Buffers + buf infrastructure | 2026-04-01 | ping/v1/ping.proto; generated via buf |
| T-005 | Valkey 9 connection pool | 2026-04-01 | pkg/cache — 3 testcontainers integration tests pass; valkey-go v1.0.73 |
| T-004 | PostgreSQL 18.3 + Atlas migrations | 2026-04-01 | pgvector, HNSW, RLS, FTS — all integration tests pass |
| T-001 | Project setup: Go module, folder structure, Makefile | 2026-03-04 | Initial scaffold |
| T-002 | Core configuration system | 2026-03-04 | pkg/config — 7 tests |
| T-003 | Structured logger | 2026-03-04 | pkg/logger — 4 tests |
| T-090 | Docker Compose dev environment | 2026-03-04 | PG18, Valkey 9, MinIO, OpenBao, Jaeger |
| T-091 | CI pipeline (GitHub Actions) | 2026-03-04 | test + lint + build + security scan |
