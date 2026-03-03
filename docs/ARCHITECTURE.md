# Infinite Brain — Architecture & AI Workflow Guide

> **AI INSTRUCTION**: This is the PRIMARY document you must read at the start of every session.
> It defines how to work in this codebase, the tech stack, coding standards, and workflow rules.
> After reading this, read `docs/TASKS.md` to understand the current project state.

---

## Workflow Protocol (AI Must Follow Every Session)

1. **Read this file first** — understand stack, patterns, and constraints
2. **Read `docs/TASKS.md`** — find in-progress or next planned tasks
3. **Read the feature spec** in `features/<task-id>-<name>/README.md` before touching code
4. **Update task status** in TASKS.md to `in_progress` before writing code
5. **Write tests first** (or alongside) — no code ships without tests
6. **Update TASKS.md** to `completed` only when all tests pass and criteria are met
7. **Never skip steps** — the documents are the source of truth

---

## Project Overview

| Property | Value |
|---|---|
| App name | Infinite Brain |
| Domain | infinitebrain.io |
| Language | Go 1.26.1 |
| Purpose | Intelligence layer that fights entropy at every scale — individual → team → org → beyond |
| Architecture | Modular monolith → microservices when justified |
| API style | connect-go: REST/HTTP-JSON + gRPC + gRPC-Web from a single handler |

---

## Technology Stack

### Backend

| Layer | Technology | Reason |
|---|---|---|
| Language | Go 1.26.1 | Latest stable; performance, simplicity, excellent concurrency |
| API Framework | connect-go (`connectrpc.com/connect`) | Single handler serves REST/HTTP-JSON + gRPC + gRPC-Web; `.proto` files are the contract; generated via Buf |
| OIDC | `github.com/coreos/go-oidc/v3` | Standard OIDC token validation; works with Zitadel, Keycloak, Okta, any OIDC provider |
| Database | PostgreSQL 18.3 + pgvector | Relational + vector embeddings; latest stable |
| Migrations | Atlas (`ariga.io/atlas`) | Schema-as-code; runs migrations in transactions; multi-branch integrity |
| ORM/Query | sqlc (`github.com/sqlc-dev/sqlc`) | Type-safe SQL, no magic, generated code |
| Cache/Pub-Sub | Valkey 9 (`github.com/valkey-io/valkey-go`) | Open source Redis fork (BSD-3); backed by Linux Foundation, AWS, Google; drop-in compatible |
| Background Jobs | River (`github.com/riverqueue/river`) | PostgreSQL-native job queue; transactional enqueue; no separate infra |
| Auth | JWT (`github.com/golang-jwt/jwt/v5`) + argon2id + pepper | Stateless auth with refresh token rotation; argon2id + server-side pepper (OWASP 2026) |
| Secrets / KMS | OpenBao (`openbao/openbao`) | Open source Vault fork (MPL-2.0); dynamic secrets, auto-rotation, encryption-as-a-service |
| Config | cleanenv (`github.com/ilyakaznacheev/cleanenv`) | Struct-based, no global state, env + yaml, minimal dependencies |
| Logger | slog (stdlib) | Go 1.21 standard library; zero dependencies; structured JSON logs |
| Validation | go-playground/validator | Struct tag validation (Huma also validates at HTTP layer) |
| Testing | testify + testcontainers-go | Unit + integration with real DBs |
| Mocking | mockery v3 | Interface mocks; 5–10× faster generation than v2 |
| AI — Claude | `github.com/anthropics/anthropic-sdk-go` | Official SDK, production-ready (v1+) |
| AI — Provider | Provider interface (T-020) + MCP adapter (T-097) | Pluggable: Claude, OpenAI, or any MCP-compatible model |
| Embeddings | OpenAI text-embedding-3-small (default) or via MCP | For semantic search via pgvector |
| Voice | OpenAI Whisper API | Audio transcription |
| MCP | `github.com/modelcontextprotocol/go-sdk` | Official Go MCP SDK |
| Observability | OpenTelemetry + Prometheus + Grafana Tempo | Traces (Tempo), metrics (Prometheus), logs (slog → Loki) |
| Email Inbound | Postal / Mailgun inbound webhooks | Email capture |
| Push Notifications | APNs (Apple Push) + FCM | iOS + Apple Watch |
| File Storage | S3-compatible (MinIO local, AWS S3 prod) | Attachments, voice files |

### Dev Infrastructure

| Tool | Purpose |
|---|---|
| Docker Compose | Local dev environment |
| GitHub Actions | CI: test, lint, security scan, build |
| golangci-lint | Static analysis (strict config) |
| Makefile | All common tasks (`make test`, `make migrate`, etc.) |

---

## Folder Structure

```
infinite_brain/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: wire up and start
├── internal/                    # Private application code
│   ├── capture/                 # Note capture domain
│   ├── ai/                      # AI engine: classify, tag, search, generate
│   ├── calendar/                # Calendar events, scheduling
│   ├── notifications/           # Push, Watch, email notifications
│   ├── auth/                    # JWT auth, session management
│   ├── contacts/                # Contacts + relationship graph
│   ├── integrations/            # Telegram, WhatsApp, Slack, GitHub, etc.
│   ├── adhd/                    # ADHD engine: timers, triage, energy
│   └── storage/                 # S3 / file storage abstraction
├── pkg/                         # Reusable packages (no business logic)
│   ├── config/                  # Config loading + validation
│   ├── logger/                  # zerolog setup
│   ├── errors/                  # Typed error handling
│   └── middleware/              # HTTP middleware (auth, logging, rate limit)
├── api/
│   └── http/                    # OpenAPI spec, handler registration
├── db/
│   ├── migrations/              # SQL migration files (goose)
│   ├── queries/                 # SQL query files (sqlc input)
│   └── sqlc/                    # Generated sqlc code (do not edit manually)
├── features/                    # Feature specs (one folder per task)
├── tests/
│   ├── integration/             # Integration tests (testcontainers)
│   └── e2e/                     # End-to-end API tests
├── docs/
│   ├── IDEA.md                  # Product vision and idea
│   ├── ARCHITECTURE.md          # THIS FILE — read first every session
│   └── TASKS.md                 # All tasks and their statuses
├── .github/
│   └── workflows/               # CI pipelines
├── docker-compose.yml           # Local dev services
├── Makefile                     # Developer commands
├── go.mod
├── go.sum
└── README.md
```

---

## Domain Model (Internal Packages)

Each internal package follows this layout:

```
internal/<domain>/
├── model.go         # Domain structs (pure Go, no DB tags)
├── repository.go    # Repository interface
├── repository_pg.go # PostgreSQL implementation
├── service.go       # Business logic service interface
├── service_impl.go  # Business logic implementation
├── handler.go       # HTTP handlers (thin, delegate to service)
├── routes.go        # Route registration
└── <domain>_test.go # Tests for all above
```

### Core Domains

| Domain | Responsibility |
|---|---|
| `capture` | Notes, voice notes, attachments, inbox |
| `ai` | Classification, tagging, embeddings, Q&A, digests |
| `calendar` | Events, calendar sync, scheduling |
| `notifications` | APNs, FCM, email, Watch haptics |
| `auth` | Users, JWT, sessions, OAuth |
| `contacts` | People, orgs, interactions, follow-ups |
| `integrations` | All third-party integrations |
| `adhd` | Focus timer, triage, energy, hyperfocus guard |
| `storage` | File uploads, S3, MinIO |

---

## Coding Standards

### General Rules

- **Idiomatic Go**: follow `go vet`, `golangci-lint`, and `Effective Go`
- **No magic**: avoid reflection-heavy frameworks; prefer explicit code
- **Error handling**: always wrap errors with context using `fmt.Errorf("...: %w", err)`
- **No panic in production code**: only `panic` in `main()` during startup validation
- **Context everywhere**: every function that does I/O takes a `context.Context` as first param
- **Interfaces at consumption point**: define interfaces where they're used, not where implemented
- **Small functions**: max ~50 lines; if longer, extract sub-functions
- **No global state**: use dependency injection (constructor functions)

### API Design

- All endpoints: `/api/v1/<resource>`
- Response envelope:
  ```json
  { "data": {...}, "meta": {...}, "error": null }
  { "data": null, "error": { "code": "NOT_FOUND", "message": "..." } }
  ```
- Use HTTP status codes correctly (200, 201, 204, 400, 401, 403, 404, 422, 500)
- Pagination: cursor-based (not offset) for all list endpoints
- Authentication: `Authorization: Bearer <jwt>` header

### Database

- All DB access through sqlc-generated code
- Never write raw SQL in handler or service layers
- Migrations in `db/migrations/`, numbered sequentially: `001_init.sql`, `002_add_users.sql`
- Every migration must be reversible (Up + Down)
- Add indexes for all FK columns and commonly filtered columns

### Testing Requirements (NON-NEGOTIABLE)

Every feature must have:

| Test Type | Requirement |
|---|---|
| Unit tests | All service and utility functions |
| Integration tests | All repository functions against real PostgreSQL (testcontainers) |
| HTTP tests | All API endpoints using `httptest` |
| Coverage | Minimum 90% per package (100% for auth/security paths) |

Test naming: `TestFunctionName_Scenario_ExpectedResult`

```go
// Good
func TestNoteService_Create_ReturnsIDOnSuccess(t *testing.T) {}
func TestNoteService_Create_FailsWhenTitleEmpty(t *testing.T) {}

// Bad
func TestCreate(t *testing.T) {}
```

Always use table-driven tests for multiple scenarios.

### Dependency Injection Pattern

```go
// Constructor injection — always
type NoteService struct {
    repo   NoteRepository
    ai     AIProvider
    logger zerolog.Logger
}

func NewNoteService(repo NoteRepository, ai AIProvider, logger zerolog.Logger) *NoteService {
    return &NoteService{repo: repo, ai: ai, logger: logger}
}
```

### Error Handling Pattern

```go
// pkg/errors — typed errors
type AppError struct {
    Code    string
    Message string
    Err     error
}

var (
    ErrNotFound   = &AppError{Code: "NOT_FOUND", Message: "resource not found"}
    ErrUnauthorized = &AppError{Code: "UNAUTHORIZED", Message: "authentication required"}
    ErrValidation = &AppError{Code: "VALIDATION_ERROR", Message: "validation failed"}
)
```

---

## AI Integration Architecture

### Provider Abstraction

```go
// internal/ai/provider.go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
}
```

All AI calls go through this interface. The concrete implementation can be swapped
(Claude → GPT-4o) without changing business logic.

### AI Processing Pipeline

```
Capture Input
    ↓
Inbox Queue (Redis/Asynq)
    ↓
AI Worker picks up job
    ↓
1. Transcribe (if audio)
2. Extract entities (people, projects, dates, topics)
3. Classify → PARA category
4. Auto-tag
5. Generate embedding (store in pgvector)
6. Link to related notes/tasks/contacts
7. Move from Inbox to destination
    ↓
Notify user (if relevant)
```

### Prompting Standards

- All AI prompts live in `internal/ai/prompts/` as Go constants or template files
- Prompts are versioned (v1, v2) — never edit in place
- Every prompt change requires a test validating output format
- Use structured output (JSON mode) for all classification tasks

---

## Background Jobs

All async work runs via Asynq workers:

| Job | Trigger | Description |
|---|---|---|
| `ai:process_capture` | On new capture | Full AI pipeline for a note |
| `ai:generate_digest` | Daily cron | Generate daily digest |
| `ai:weekly_review` | Weekly cron | Generate weekly review |
| `notification:send` | On event | Route and deliver notification |
| `integration:sync` | Scheduled | Sync calendar, Readwise, etc. |
| `adhd:check_focus` | Every 5 min | Check if focus timer running too long |

---

## Environment Variables

All config via env variables (12-factor). See `configs/example.env` for full list.

Required at startup:
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_URL` — Redis connection string
- `JWT_SECRET` — JWT signing key (min 32 chars)
- `ANTHROPIC_API_KEY` — Claude API key
- `OPENAI_API_KEY` — OpenAI API key (Whisper, embeddings)
- `S3_BUCKET` — File storage bucket
- `S3_ENDPOINT` — S3 endpoint (AWS or MinIO)

---

## Git Workflow

- Branch naming: `feature/T-XXX-short-description`, `fix/T-XXX-short-description`
- Commit messages: `[T-XXX] verb: short description`
  - Example: `[T-010] feat: add note CRUD API with validation`
- No direct commits to `main`
- PRs require passing CI (tests + lint)
- Squash merge to keep history clean

---

## Making Changes Checklist

Before marking any task complete:

- [ ] Feature spec in `features/<id>/README.md` is updated
- [ ] All acceptance criteria in the spec are met
- [ ] Unit tests written and passing (`make test`)
- [ ] Integration tests written and passing (`make test-integration`)
- [ ] Linter passes (`make lint`)
- [ ] TASKS.md updated to `completed`
- [ ] Migration files added (if schema changed)
- [ ] OpenAPI spec updated (if API changed)
- [ ] No secrets committed (check with `make security-scan`)
