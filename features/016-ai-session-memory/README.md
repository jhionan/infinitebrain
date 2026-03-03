# T-016 — AI Session Memory

## Overview

Persist AI reasoning across sessions. When a Claude session ends, what was reasoned, inferred, and discovered should survive. The next session loads relevant memories as context, giving the AI continuity without the user having to re-explain everything.

This is distinct from user-captured notes. Notes are what the user writes. Agent memories are what the AI observes and reasons during a session.

---

## Why

The current plan has no persistence of AI reasoning. Every session starts cold. For a second brain that's supposed to know you deeply, that's a fundamental gap. The AI should get smarter about you over time — remembering what projects you were working on, what decisions were made, what context was established.

Parallel agents also need a shared context surface. When two agents work on the same project simultaneously, they should be able to read each other's reasoning. The database is the shared memory bus.

---

## Schema

```sql
CREATE TABLE agent_memories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id  UUID NOT NULL,
    agent_id    TEXT NOT NULL,              -- e.g. "classify-worker", "digest-worker", "qa-agent"
    project_id  UUID REFERENCES nodes(id),  -- nullable: personal memories have no project scope
    type        TEXT NOT NULL,              -- 'observation' | 'decision' | 'context' | 'pattern' | 'error'
    content     TEXT NOT NULL,
    embedding   VECTOR(1536),
    confidence  FLOAT DEFAULT 1.0,          -- 0.0–1.0: how confident the AI is in this memory
    metadata    JSONB DEFAULT '{}',
    expires_at  TIMESTAMPTZ,               -- nullable: some memories are permanent, others ephemeral
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON agent_memories (user_id, project_id);
CREATE INDEX ON agent_memories (user_id, type);
CREATE INDEX ON agent_memories (session_id);
CREATE INDEX ON agent_memories USING hnsw (embedding vector_cosine_ops);
```

### Memory Types

| Type | Description | Example |
|---|---|---|
| `observation` | Something the AI noticed | "User tends to capture ideas late at night" |
| `decision` | A conclusion reached | "This project uses CQRS pattern" |
| `context` | Background established | "Project X is a payments API in Go" |
| `pattern` | Recurring behavior | "User always skips exercise chunks on Monday" |
| `error` | Something that went wrong | "Classification failed for voice notes with background noise" |

---

## API Endpoints

All under `/api/v1/memories`.

### Store a memory (internal — called by AI workers, not user-facing)
```
POST /api/v1/memories
Body: { agent_id, session_id, project_id, type, content, confidence, metadata, expires_at }
```

### Query memories for context loading
```
GET /api/v1/memories?project_id=&type=&limit=20
```

### Semantic search over memories
```
POST /api/v1/memories/search
Body: { query: "payments project decisions", project_id, limit: 10 }
Returns: memories ranked by embedding similarity
```

### Delete a memory (user can curate)
```
DELETE /api/v1/memories/:id
```

---

## Context Loading at Session Start

When an AI worker starts (e.g., the Q&A worker receives a question), it loads context:

1. Fetch last 10 `context` + `decision` memories for the relevant `project_id`
2. Fetch top 5 semantically similar memories to the current query
3. Inject into the system prompt as structured context

```go
// internal/ai/context_loader.go
type ContextLoader struct {
    repo AgentMemoryRepository
}

func (c *ContextLoader) Load(ctx context.Context, userID, projectID uuid.UUID, query string) ([]AgentMemory, error)
```

---

## Shared Memory for Parallel Agents

No special coordination needed. Multiple Asynq workers write to `agent_memories` concurrently. PostgreSQL handles concurrent inserts safely. Workers read with `SELECT ... WHERE project_id = $1 ORDER BY created_at DESC LIMIT 20`.

The database is the shared memory bus.

---

## Acceptance Criteria

- [ ] `agent_memories` table created via migration
- [ ] POST /api/v1/memories stores a memory with embedding
- [ ] GET /api/v1/memories returns memories filtered by project and type
- [ ] POST /api/v1/memories/search returns semantically similar memories
- [ ] DELETE /api/v1/memories/:id removes a memory
- [ ] `ContextLoader` loads relevant memories before AI worker runs
- [ ] Two concurrent workers can write memories to the same project without conflict
- [ ] Memories with `expires_at` in the past are excluded from queries
- [ ] Unit tests for ContextLoader
- [ ] Integration tests for all endpoints against real PostgreSQL
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL + migrations) — must be complete
- T-020 (AI provider abstraction) — needed for embedding generation
- T-023 (Semantic search) — shares pgvector infrastructure

## Notes

- `session_id` is generated per HTTP request or Asynq job — not a persistent user session
- Memories are user-owned data; full deletion on user account deletion (CASCADE)
- `confidence` allows the AI to flag uncertain memories — low confidence memories are deprioritized in context loading
