# T-048 — Daily Chunk Planner

## Overview

A time-boxed daily planning system built for ADHD. The day is divided into N chunks (default 16). Each chunk has a type (work, chore, exercise, personal, free). Order does not matter — the user picks the next chunk based on current energy. The system enforces 100% focus during a chunk and notifies when the timer ends.

Replaces and supersedes T-040 (Focus timer) and T-047 (Daily planning assistant).

---

## Why

Traditional to-do lists overwhelm. Pomodoro timers are too rigid. This system removes both problems:
- No overwhelming list of 40 tasks — just N chunks to fill
- No fixed schedule — pick what fits your energy right now
- Completion is the unit of success, not "the right task at the right time"

For ADHD: small wins every chunk, no decision paralysis on task order, clear "what's next?" signal.

---

## Core Flow

```
1. Morning: confirm today's chunk mix (or auto-accept template)
   └── 16 chunks: 8× work, 3× chore, 1× exercise, 2× personal, 2× free

2. Pick any pending chunk from the available pool

3. If work chunk → "What are you working on?"
   └── AI suggests top 3 tasks (priority + energy + context)
   └── User picks, types custom, or skips (starts unfocused work session)

4. Timer starts → full focus

5. Timer ends → notification: "Chunk complete. Pick your next one."

6. Repeat until all 16 done or day ends
```

---

## Schema

```sql
-- Reusable day template
CREATE TABLE chunk_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,          -- 'weekday', 'weekend', 'light day'
    chunk_min   INT NOT NULL DEFAULT 60,
    slots       JSONB NOT NULL,         -- [{"type":"work","count":8},{"type":"chore","count":3},...]
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Daily plan instance (one per day per user)
CREATE TABLE daily_plans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_id  UUID REFERENCES chunk_templates(id),
    date         DATE NOT NULL,
    chunk_min    INT NOT NULL DEFAULT 60, -- override template default
    status       TEXT NOT NULL DEFAULT 'active', -- 'active' | 'completed' | 'abandoned'
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, date)
);

-- Individual chunk within a plan
CREATE TABLE chunks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id       UUID NOT NULL REFERENCES daily_plans(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          TEXT NOT NULL,        -- 'work' | 'chore' | 'exercise' | 'personal' | 'free'
    node_id       UUID REFERENCES nodes(id), -- linked task/node (nullable — set when chunk starts)
    title         TEXT,                 -- what user said they'd work on
    duration_min  INT NOT NULL,         -- actual duration for this chunk
    status        TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'active' | 'completed' | 'skipped'
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    sequence      INT,                  -- order in which it was executed (set on completion)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON daily_plans (user_id, date);
CREATE INDEX ON chunks (plan_id, status);
CREATE INDEX ON chunks (user_id, status);
```

### Chunk Types

| Type | Description |
|---|---|
| `work` | Professional or project work — AI suggests a task at start |
| `chore` | Household or admin task — no AI suggestion, free-form |
| `exercise` | Physical activity |
| `personal` | Non-work, non-chore personal time (calls, errands) |
| `free` | Unstructured — rest, spontaneous, no tracking |

---

## API Endpoints

### Templates

```
POST   /api/v1/chunk-templates          Create a template
GET    /api/v1/chunk-templates          List user's templates
PUT    /api/v1/chunk-templates/:id      Update template
DELETE /api/v1/chunk-templates/:id      Delete template
```

### Daily Plan

```
POST /api/v1/daily-plans                Create today's plan (from template or custom)
GET  /api/v1/daily-plans/today          Get today's plan with all chunks
GET  /api/v1/daily-plans/:date          Get plan for a specific date
```

### Chunks

```
GET  /api/v1/daily-plans/today/chunks              List all chunks (filter by status)
POST /api/v1/chunks/:id/start                      Start a chunk (sets status=active, started_at=now)
POST /api/v1/chunks/:id/complete                   Complete a chunk (sets status=completed, completed_at=now)
POST /api/v1/chunks/:id/skip                       Skip a chunk
POST /api/v1/chunks/:id/suggest                    Get AI task suggestions for a work chunk
```

### Suggest endpoint (work chunks only)

```
POST /api/v1/chunks/:id/suggest
Response: {
  "suggestions": [
    { "node_id": "...", "title": "Implement validateTransfer()", "reason": "highest priority + you worked on this yesterday" },
    { "node_id": "...", "title": "Review PR #42", "reason": "deadline today" },
    { "node_id": "...", "title": "Write unit tests for auth", "reason": "blocked by nothing, medium energy task" }
  ]
}
```

---

## Start Chunk Flow (Server Side)

```go
func (s *ChunkService) Start(ctx context.Context, chunkID uuid.UUID, nodeID *uuid.UUID, title *string) (*Chunk, error) {
    // 1. Verify no other chunk is active for this user today
    // 2. Set chunk status = active, started_at = now()
    // 3. If nodeID provided, link to node
    // 4. Start timer (Asynq delayed job: notify after duration_min)
    // 5. Return chunk
}
```

### Timer via Asynq

```go
// On chunk start, enqueue a delayed notification job
task := asynq.NewTask("notification:chunk_complete", payload,
    asynq.ProcessAt(time.Now().Add(time.Duration(chunk.DurationMin)*time.Minute)),
    asynq.TaskID(fmt.Sprintf("chunk-timer-%s", chunkID)), // deduplicate
)
```

When the job fires:
1. Mark chunk as `completed`
2. Fire notification: "Chunk done. Pick your next one." (via bot/push)
3. Update `sequence` based on completed count for the day

---

## AI Task Suggestion Logic

For the `/suggest` endpoint on a work chunk:

```go
func (s *ChunkService) Suggest(ctx context.Context, userID uuid.UUID) ([]TaskSuggestion, error) {
    // 1. Fetch top 10 pending tasks ordered by priority score
    // 2. Load last 3 work chunks completed today (context: avoid repeating same task area)
    // 3. Load agent memories for context (T-016): what was worked on recently?
    // 4. Ask AI to rank and pick top 3 with reasons
}
```

AI prompt returns structured JSON: `[{ node_id, title, reason }]`.

---

## Plan Generation

```
POST /api/v1/daily-plans
Body: { "template_id": "...", "date": "2026-04-01" }  // template_id optional
```

If no `template_id` → use user's default template.
If no default template → create a standard weekday plan (8 work, 3 chore, 1 exercise, 2 personal, 2 free).

Chunks are created from `slots` in the template: for each `{ type, count }` pair, insert `count` chunk rows with `status = pending`.

---

## End of Day

If not all chunks are completed when day ends (midnight):
- Remaining `pending` chunks are set to `skipped`
- Plan status → `abandoned` if < 50% complete, `completed` otherwise
- A summary is included in next morning's digest: "Yesterday: 11/16 chunks. 3 work, 2 chore, 1 exercise."

---

## Acceptance Criteria

- [ ] `chunk_templates`, `daily_plans`, `chunks` tables created via migration
- [ ] Create/read/update/delete for templates
- [ ] POST /api/v1/daily-plans generates correct chunk rows from template
- [ ] Only one active chunk allowed per user at a time (enforced at service layer)
- [ ] POST /api/v1/chunks/:id/start sets active status, enqueues Asynq timer
- [ ] Timer fires notification on completion
- [ ] POST /api/v1/chunks/:id/complete marks complete, records sequence
- [ ] POST /api/v1/chunks/:id/suggest returns 3 AI-ranked task suggestions for work chunks
- [ ] End-of-day job cleans up incomplete plans
- [ ] Daily digest includes yesterday's chunk summary
- [ ] Unit tests for ChunkService (start, complete, skip, suggest logic)
- [ ] Integration tests for all endpoints
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL)
- T-006 (HTTP server)
- T-007 (Auth)
- T-016 (AI session memory — context for suggestions)
- T-020 (AI provider — task suggestion prompt)
- T-025 (Daily digest — chunk summary)
- T-028 (Knowledge graph — node_id FK on chunks)
- T-030 (Tasks — the nodes being suggested)

## Notes

- Chunk duration is per-chunk (can differ within a plan). Template sets the default. User can override individual chunks before starting.
- `free` chunks are never tracked — starting them immediately marks them active with no notification on end (user controls when it's over).
- Future: energy-awareness (T-043) feeds into suggestion ranking — high-complexity tasks suggested only during high-energy chunks.
