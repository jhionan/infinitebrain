# Feature: Note Capture — Core CRUD API

**Task ID**: T-010
**Status**: planned
**Epic**: Capture Engine

## Goal

Implement the core note domain: the fundamental unit of capture in Infinite Brain.
A note is any text content captured by the user, with optional metadata.
All notes enter the system via the Inbox and are later classified by AI.

## Acceptance Criteria

- [ ] `internal/capture/model.go` — Note struct with all fields
- [ ] `internal/capture/repository.go` — NoteRepository interface
- [ ] `internal/capture/repository_pg.go` — PostgreSQL implementation (sqlc)
- [ ] `internal/capture/service.go` — NoteService interface
- [ ] `internal/capture/service_impl.go` — Business logic
- [ ] `internal/capture/handler.go` — HTTP handlers
- [ ] `internal/capture/routes.go` — Route registration
- [ ] `db/migrations/002_create_notes.sql` — Notes table migration
- [ ] `db/queries/notes.sql` — sqlc query definitions
- [ ] All endpoints require authentication (JWT)
- [ ] Unit tests for service layer (mocked repository)
- [ ] Integration tests for repository layer (real PostgreSQL via testcontainers)
- [ ] HTTP tests for all endpoints (httptest)
- [ ] Coverage ≥ 80%

## Data Model

Notes are a projection of `node` rows where `type = 'note'`.
The canonical schema lives in **T-004** (`features/004-database/README.md`).

```go
type Note struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    UserID       uuid.UUID
    Title        string           // optional, AI can generate
    Content      string           // markdown; empty when is_phi = true (use ContentEnc)
    Source       NoteSource       // manual | voice | email | telegram | whatsapp | webhook
    Status       NoteStatus       // inbox | classified | archived
    PARACategory PARACategory     // project | area | resource | archive | nil (in inbox)
    ProjectID    *uuid.UUID       // link to project node if classified
    Tags         []string
    Attachments  []Attachment
    Visibility   Visibility       // personal | team | org | public — default: personal
    IsPHI        bool             // if true: Content is encrypted in DB
    CreatedAt    time.Time
    UpdatedAt    time.Time
    ArchivedAt   *time.Time
}

type NoteSource string
const (
    SourceManual   NoteSource = "manual"
    SourceVoice    NoteSource = "voice"
    SourceEmail    NoteSource = "email"
    SourceTelegram NoteSource = "telegram"
    SourceWhatsApp NoteSource = "whatsapp"
    SourceWebhook  NoteSource = "webhook"
)

type Visibility string
const (
    VisibilityPersonal Visibility = "personal"
    VisibilityTeam     Visibility = "team"
    VisibilityOrg      Visibility = "org"
    VisibilityPublic   Visibility = "public"
)
```

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/notes` | Create a note (lands in Inbox) |
| `GET` | `/api/v1/notes` | List notes (paginated, cursor-based) |
| `GET` | `/api/v1/notes/:id` | Get a single note |
| `PATCH` | `/api/v1/notes/:id` | Update title, content, tags |
| `DELETE` | `/api/v1/notes/:id` | Soft-delete a note |
| `GET` | `/api/v1/inbox` | List unclassified notes (Inbox) |
| `POST` | `/api/v1/notes/:id/archive` | Archive a note |

## Create Note Request

```json
{
  "title": "Optional title",
  "content": "The actual note content",
  "source": "manual",
  "tags": ["idea", "work"]
}
```

## Business Rules

1. New notes always start with `status: inbox`
2. `title` is optional — AI will generate one during classification
3. `content` is required and must not be empty
4. `source` defaults to `manual` if not provided
5. Soft-delete: notes are marked `archived_at`, never hard-deleted
6. After creation, an AI processing job is enqueued via Asynq

## Database Schema

Notes are rows in the `nodes` table (T-004) with `type = 'note'`.
There is no separate `notes` table. The `source` and `status` fields live in `metadata JSONB`.

```sql
-- No separate notes table. Query pattern:
SELECT * FROM nodes
WHERE org_id = $1
  AND user_id = $2
  AND type = 'note'
  AND visibility = ANY($3)   -- visibility scope from auth context
  AND deleted_at IS NULL;

-- metadata contains note-specific fields:
-- { "source": "telegram", "status": "inbox" }
```

Full schema in `db/schema/002_nodes.hcl` (T-004). Queries in `db/queries/notes.sql`.
