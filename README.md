# Infinite Brain

> Entropy is the default. Infinite Brain is the energy you inject to keep things in balance.

**infinitebrain.io** — AGPL-3.0 — Go 1.26 · PostgreSQL 18 · pgvector · Valkey · OpenBao

---

## What It Is

You forget things. Your team forgets things. Your company forgets why it made decisions,
loses context when people leave, and slowly drifts from its own direction.
This is not a people problem. It is a systems problem.
At every scale — individual, team, squad, company — the natural state is entropy.

Infinite Brain is the intelligence layer that fights that entropy.
One engine. One data model. Applied recursively at every level of any hierarchy.

Everything you capture — notes, decisions, tasks, emails, voice memos, code context —
becomes part of a knowledge graph that understands relationships, surfaces what matters,
and keeps you aligned with where you're actually trying to go.

The same system that helps an individual stay focused helps a team stay coherent
and helps an organization stay true to its direction. It scales because the model
is recursive, not because it was bolted together from separate products.

> Infinite Brain is built with Infinite Brain.
> Every feature spec, architecture decision, and task in this repository
> was captured, organized, and planned inside the product itself.
> See [docs/META.md](docs/META.md) for the self-building loop.

---

## What It Can Be Used For

### For Individuals — your external brain

You have more context in your head than any tool captures.
Infinite Brain changes that.

- Capture notes, voice memos, and ideas from anywhere — email, WhatsApp, web clipper, API
- Ask questions across everything you've ever saved: *"What did I decide about X last year?"*
- See connections between ideas you never linked yourself
- Stay focused: task decomposition, priority scoring, and a system that knows when you're overloaded
- Every decision you make becomes part of a searchable, queryable history

**Who it's for**: founders, engineers, researchers, anyone who thinks in systems and loses too much to memory.

---

### For Teams — shared knowledge that outlives any one person

When your best engineer leaves, what stays behind?
Infinite Brain keeps the context, not just the artifacts.

- Shared knowledge graph across the whole team — decisions, ADRs, context, tribal knowledge
- New team members get up to speed from the system, not from whoever has time to explain things
- Every task carries the reasoning behind it — not just what was done, but why
- Business rules are explicit, versioned, and injected into every agent and AI task automatically
- Integrates with your existing tools: Jira, Linear, Asana, GitHub Projects — IB stays the source of truth

**Who it's for**: engineering teams, product squads, any group where context loss is a real cost.

---

### For Organizations — collective intelligence without surveillance

Most org-level intelligence tools require trading privacy for insight.
Infinite Brain is designed so you don't have to make that trade.

- The organization learns from its people — without identifying or benchmarking individuals
- Ideas surface through merit, not through who said them (no HiPPO effect)
- Every major decision is scored against the organization's True North — its honest objectives
- Coherence score across the whole org: are we actually moving in the direction we say we are?
- Privacy by architecture: k-anonymity enforced at the query layer, not as a policy

**Who it's for**: CTOs, COOs, chiefs of staff — anyone responsible for organizational direction.

---

### As a Platform — the intelligent context layer for AI agents

AI tools are only as good as the context they have.
Infinite Brain gives them the full picture.

- MCP server: any AI tool (Cursor, Claude Code, Copilot) can query your knowledge graph in real time
- Business rules and requirements are automatically injected into every agent task
- Agent tasks are decomposed from goals, verified against acceptance criteria, and tracked
- Build your own integrations on top of the context API — one query returns everything relevant to a topic
- SOC2 + HIPAA + EU AI Act compliance built in — safe to use for regulated industries

**Who it's for**: teams building AI-assisted workflows, developers using LLM tools daily, regulated industries that need auditability.

---

## The Architecture

```
cmd/server/              → Entry point, wiring
internal/                → Business logic, layered by domain
  capture/               → Notes, voice, email, webhooks
  ai/                    → Provider abstraction, classify, tag, Q&A, agents
  auth/                  → JWT, OIDC, RBAC
  nodes/                 → Universal knowledge graph (nodes + edges)
  org/                   → Multi-tenancy, org units, team brains
  security/              → PromptGuard, anomaly detection, honeypot
  compliance/            → Audit log, PHI encryption, AI usage register
pkg/                     → Zero-business-logic utilities (config, logger, errors)
db/
  schema/                → Atlas HCL schema files (source of truth)
  queries/               → sqlc SQL input
  sqlc/                  → Generated type-safe query code
features/                → Detailed spec for every task (read before implementing)
docs/                    → Architecture, tasks, ideas, brainstorm
```

**Layer rule** (non-negotiable): `Handler → Service → Repository → Database`.
No SQL in services. No business logic in handlers. No layer skipping.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26.1 |
| API | connect-go (Buf) — REST + gRPC + gRPC-Web from one handler |
| Database | PostgreSQL 18.3 + pgvector (HNSW) |
| Migrations | Atlas (schema-as-code HCL) |
| Query layer | sqlc (type-safe generated code) |
| Cache / pub-sub | Valkey 9 (open-source Redis fork, Linux Foundation) |
| Background jobs | River (PostgreSQL-native, transactional enqueue) |
| Auth | JWT + argon2id + server-side pepper |
| Secrets / KMS | OpenBao (open-source Vault fork, MPL-2.0) |
| AI | Claude (Anthropic) — provider interface; pluggable |
| Embeddings | text-embedding-3-small (via provider interface) |
| Voice | OpenAI Whisper |
| MCP | go-sdk (official) |
| Compliance | SOC2 + HIPAA + EU AI Act + GDPR |

---

## Quick Start

```bash
# 1. Start infrastructure (PostgreSQL 18, Valkey, OpenBao)
make docker-up

# 2. Configure environment
cp configs/example.env .env
# Edit .env — add ANTHROPIC_API_KEY and other keys

# 3. Apply schema migrations
make db-migrate

# 4. Run the server
make run
```

Server: `http://localhost:8080` · Health: `GET /health`

---

## Developer Commands

```bash
make test                        # Unit tests
make test-integration            # Integration tests (requires Docker)
make test-coverage               # Coverage HTML report
make lint                        # golangci-lint
make build                       # Build binary

make db-migrate                  # Apply Atlas migrations
make db-diff                     # Show schema drift
make db-generate                 # Regenerate sqlc code
make proto                       # Regenerate connect-go from .proto files

make docker-up                   # Start PostgreSQL, Valkey, OpenBao
make docker-down                 # Stop services

make security-scan               # gosec
make vuln-check                  # govulncheck

make help                        # Full target list
```

---

## Documentation

| Document | Purpose |
|---|---|
| [CLAUDE.md](CLAUDE.md) | Coding standards, architecture rules, security requirements — **read first** |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Tech stack decisions and patterns |
| [docs/IDEA.md](docs/IDEA.md) | Product vision, personas, business model |
| [docs/TASKS.md](docs/TASKS.md) | All tasks, phases, and current status |
| [docs/META.md](docs/META.md) | How IB is used to build IB (the self-building loop) |
| [docs/BRAINSTORM.md](docs/BRAINSTORM.md) | Living ideas document |
| [features/](features/) | Detailed spec per feature — read before implementing |

---

## Contributing

1. Read [CLAUDE.md](CLAUDE.md) — understand the standards before touching code
2. Read [docs/TASKS.md](docs/TASKS.md) — find a `planned` task
3. Read `features/<task-id>/README.md` — understand the full spec
4. Create a branch: `feature/T-XXX-short-description`
5. Write tests alongside code (not after)
6. Open PR — CI must pass: tests + lint + 90% coverage

All AI inputs pass through `PromptGuard.Sanitize`. All external content is untrusted.
See [features/177-prompt-guard/](features/177-prompt-guard/) for the security model.

---

## Security

- Responsible disclosure: open a private GitHub security advisory
- Known attack surface: [features/177-prompt-guard/](features/177-prompt-guard/) (prompt injection),
  [features/099-honeypot/](features/099-honeypot/) (honeypot endpoints),
  [features/130-canary-tokens/](features/130-canary-tokens/) (canary credentials)
- Compliance: SOC2 + HIPAA + EU AI Act — see [features/104-compliance/](features/104-compliance/)

---

## License

[AGPL-3.0](LICENSE) — open source, self-hostable.

Self-hosted deployments are free. The managed cloud at **infinitebrain.io** provides
zero-config hosting, automatic updates, and handles compliance overhead.

Self-hosters who deploy for their organization and use org-intelligence features for
employment decisions become providers under EU AI Act Regulation 2024/1689 and bear
applicable compliance obligations.
