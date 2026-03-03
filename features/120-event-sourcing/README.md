# T-120 — Event Sourcing

## Overview

Replace mutable state storage with an immutable event log. Every change to every aggregate
(Node, Task, Capture, Session) is stored as an event. Current state is a projection derived
by replaying events. State is never overwritten — only new events are appended.

This is the most architecturally significant decision in the project. It must be implemented
before T-028 (knowledge graph), T-010 (notes), and T-030 (tasks) — those become projections,
not the primary store.

---

## Why Event Sourcing for a Second Brain

A second brain is fundamentally about the history of thought, not just current state.

| Question | Without ES | With ES |
|---|---|---|
| What did I know about X on March 1st? | Impossible | Replay events to that timestamp |
| How did this idea evolve? | Gone — overwritten | Full event timeline per node |
| Undo this classification change | Not possible | Replay without that event |
| Why did the AI tag this as "health"? | No record | `ai.node.tagged` event with prompt + response |
| Parallel agents conflicting | Race condition | Optimistic concurrency on event version |
| HIPAA audit trail | Separate table to maintain | The event log IS the audit trail |

The event log is the source of truth. Every other table is a read-model (projection) that
can be dropped and rebuilt by replaying events.

---

## Core Concepts

```
Command → Aggregate → Events → EventStore → Projectors → Read Models
                                    │
                                    └── River async projectors (eventually consistent)
                                    └── Sync projectors (strongly consistent, for critical paths)
```

- **Command**: intent ("create this note", "tag this node")
- **Aggregate**: domain object that validates the command and raises events
- **Event**: immutable fact that something happened ("NoteCreated", "NodeTagged")
- **EventStore**: append-only PostgreSQL table
- **Projector**: subscribes to events, updates read models (nodes table, edges table, etc.)
- **Read Model**: the queryable projection (what SQL queries hit)

---

## Event Store Schema

```sql
CREATE TABLE domain_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id       UUID NOT NULL,          -- aggregate ID (node_id, task_id, org_id, etc.)
    stream_type     TEXT NOT NULL,          -- 'node' | 'task' | 'capture' | 'session' | 'org'
    event_type      TEXT NOT NULL,          -- 'node.created' | 'node.tagged' | etc.
    event_version   INT  NOT NULL,          -- monotonic per stream — optimistic concurrency
    schema_version  INT  NOT NULL DEFAULT 1,-- for event upcasting (schema evolution)
    payload         JSONB NOT NULL,         -- event data
    metadata        JSONB NOT NULL DEFAULT '{}', -- actor_id, ip, user_agent, trace_id
    org_id          UUID NOT NULL,          -- multi-tenancy; RLS by org
    causation_id    UUID,                   -- ID of event that caused this one (event chains)
    correlation_id  UUID,                   -- request/saga/session ID for distributed tracing
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary read path: load all events for an aggregate in order
CREATE UNIQUE INDEX ON domain_events (stream_id, event_version);

-- Query: all events for an org in time range (compliance, digest, insights)
CREATE INDEX ON domain_events (org_id, created_at DESC);

-- Query: events by type (projector catch-up, analytics)
CREATE INDEX ON domain_events (stream_type, event_type, created_at DESC);

-- Query: trace a causal chain
CREATE INDEX ON domain_events (correlation_id);

-- Append-only: application role has INSERT only — no UPDATE, no DELETE
-- REVOKE UPDATE, DELETE ON domain_events FROM infinitebrain_app;
```

### Optimistic Concurrency

`event_version` is monotonic per `stream_id`. When appending, include the expected version:

```sql
-- This fails with a unique constraint violation if another writer beat us
INSERT INTO domain_events (stream_id, event_version, ...)
VALUES ($1, $expected_version + 1, ...)
```

Concurrent writes to the same aggregate → one succeeds, one retries. No locks needed.

---

## Event Catalog

Every event has a stable type name (`stream_type.verb`) and a versioned JSON schema.

### Node Events
```
node.created          — title, content, source, org_id, actor_id
node.content_updated  — new_content, previous_hash (SHA-256 of old content)
node.classified       — para, confidence, model, prompt_version
node.tagged           — tags_added, tags_removed, source ('ai' | 'user')
node.linked           — to_node_id, relation_type, confidence, created_by
node.unlinked         — to_node_id, relation_type, reason
node.reviewed         — stage_from, stage_to, response ('yes' | 'no' | 'skip')
node.expired          — final_stage, reason
node.phi_flagged      — detected_by, confidence
node.deleted          — reason, actor_type
```

### Capture Events
```
capture.received      — source, content_hash, raw_size, inbound_channel
capture.transcribed   — transcript, model, latency_ms, confidence
capture.processed     — node_id, pipeline_stages_completed
capture.failed        — stage, error_code, retry_count
```

### Task Events
```
task.created          — title, linked_node_id, estimated_chunks
task.priority_scored  — score, factors, model, prompt_version
task.started          — chunk_id, context_loaded (bool)
task.completed        — actual_chunks, notes
task.snoozed          — until, reason
```

### Session / ADHD Events
```
chunk.started         — chunk_type, task_id, scheduled_duration_min
chunk.completed       — actual_duration_min, task_completed (bool)
chunk.interrupted     — reason, distraction_captured (bool)
focus.hyperfocus_detected  — duration_min, task_id
```

### Auth / Org Events
```
user.registered       — email_hash (not plaintext), auth_provider
user.login            — ip_hash, device_fingerprint, mfa_used
user.login_failed     — ip_hash, attempt_count
org.created           — plan, auto_created (bool)
member.invited        — role, invited_by
member.role_changed   — from_role, to_role, changed_by
key.rotated           — key_id, algorithm, rotated_by
```

### AI Events
```
ai.completion         — model, operation, input_tokens, output_tokens, latency_ms, cost_usd
ai.memory.stored      — agent_id, memory_type, confidence
ai.insight.generated  — from_node_id, to_node_id, similarity, validated (bool)
ai.cost.threshold     — user_id, daily_cost_usd, threshold_usd
```

---

## Aggregate Pattern

```go
// internal/eventsource/aggregate.go

// Aggregate is embedded in every domain aggregate.
type Aggregate struct {
    id      uuid.UUID
    version int           // current persisted version
    pending []DomainEvent // events raised but not yet persisted
}

func (a *Aggregate) ID() uuid.UUID      { return a.id }
func (a *Aggregate) Version() int       { return a.version }
func (a *Aggregate) Pending() []DomainEvent { return a.pending }
func (a *Aggregate) ClearPending()      { a.pending = nil }

func (a *Aggregate) raise(e DomainEvent) {
    a.pending = append(a.pending, e)
}
```

```go
// internal/capture/aggregate.go

type NodeAggregate struct {
    eventsource.Aggregate

    Title    string
    Content  string
    Tags     []string
    Para     string
    IsPHI    bool
    IsDeleted bool
}

// Commands mutate through events — never directly
func (n *NodeAggregate) Create(cmd CreateNodeCommand) error {
    if cmd.Title == "" {
        return apperrors.ErrValidation.WithDetail("title is required")
    }
    n.raise(NodeCreatedEvent{
        Title:   cmd.Title,
        Content: cmd.Content,
        Source:  cmd.Source,
    })
    return nil
}

func (n *NodeAggregate) Tag(cmd TagNodeCommand) error {
    if n.IsDeleted {
        return apperrors.ErrNotFound
    }
    n.raise(NodeTaggedEvent{
        TagsAdded:   cmd.Tags,
        TagsRemoved: cmd.RemoveTags,
        Source:      cmd.Source,
    })
    return nil
}

// Apply reconstructs state from events — called on both raise and load
func (n *NodeAggregate) Apply(e DomainEvent) {
    switch evt := e.(type) {
    case NodeCreatedEvent:
        n.Title = evt.Title
        n.Content = evt.Content
    case NodeTaggedEvent:
        n.Tags = applyTagDiff(n.Tags, evt.TagsAdded, evt.TagsRemoved)
    case NodeClassifiedEvent:
        n.Para = evt.Para
    case NodeDeletedEvent:
        n.IsDeleted = true
    }
}
```

---

## Event Store Interface

```go
// internal/eventsource/store.go

type EventStore interface {
    // Append writes events for an aggregate. Fails with ErrConcurrencyConflict
    // if expectedVersion does not match the current stream version.
    Append(ctx context.Context, streamID uuid.UUID, expectedVersion int, events []DomainEvent) error

    // Load returns all events for a stream in version order.
    Load(ctx context.Context, streamID uuid.UUID) ([]DomainEvent, error)

    // LoadFrom returns events after a given version (for catch-up projectors).
    LoadFrom(ctx context.Context, streamID uuid.UUID, fromVersion int) ([]DomainEvent, error)

    // LoadAt returns events up to a given timestamp (temporal queries).
    LoadAt(ctx context.Context, streamID uuid.UUID, at time.Time) ([]DomainEvent, error)

    // Subscribe returns a channel of new events matching the filter.
    // Used by async projectors. Events are delivered at-least-once.
    Subscribe(ctx context.Context, filter EventFilter) (<-chan DomainEvent, error)
}
```

One implementation: `PostgresEventStore`. No other implementations needed.

---

## Repository Pattern with Event Sourcing

```go
// internal/capture/repository.go

type NodeRepository interface {
    Save(ctx context.Context, node *NodeAggregate) error
    Load(ctx context.Context, id uuid.UUID) (*NodeAggregate, error)
    LoadAt(ctx context.Context, id uuid.UUID, at time.Time) (*NodeAggregate, error)  // ← time travel
}

// internal/capture/repository_es.go

type EventSourcedNodeRepo struct {
    store eventsource.EventStore
}

func (r *EventSourcedNodeRepo) Save(ctx context.Context, node *NodeAggregate) error {
    if len(node.Pending()) == 0 {
        return nil
    }
    err := r.store.Append(ctx, node.ID(), node.Version(), node.Pending())
    if err != nil {
        return fmt.Errorf("saving node events: %w", err)
    }
    node.ClearPending()
    return nil
}

func (r *EventSourcedNodeRepo) Load(ctx context.Context, id uuid.UUID) (*NodeAggregate, error) {
    events, err := r.store.Load(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("loading node events: %w", err)
    }
    if len(events) == 0 {
        return nil, apperrors.ErrNotFound
    }
    node := &NodeAggregate{}
    for _, e := range events {
        node.Apply(e)
        node.Aggregate.version = e.Version
    }
    return node, nil
}
```

---

## Projectors — Building Read Models

Projectors listen to events and update read models. Two modes:

**Sync projector** — runs in the same transaction as the event append. Used for:
- The `nodes` read-model table (queries need to be immediately consistent)
- The `edges` table
- The `audit_log` table (T-104)

**Async projector** — River job triggered by event. Used for:
- Embeddings generation (slow — calls OpenAI)
- Insight linker trigger
- Notification dispatch
- AI cost aggregation

```go
// internal/eventsource/projector.go

type Projector interface {
    // EventTypes returns the event types this projector handles.
    EventTypes() []string

    // Project updates the read model for the given event.
    Project(ctx context.Context, event DomainEvent) error
}

// internal/capture/node_projector.go

type NodeProjector struct {
    db *pgxpool.Pool
}

func (p *NodeProjector) EventTypes() []string {
    return []string{
        "node.created", "node.content_updated", "node.classified",
        "node.tagged", "node.reviewed", "node.deleted",
    }
}

func (p *NodeProjector) Project(ctx context.Context, event DomainEvent) error {
    switch evt := event.(type) {
    case NodeCreatedEvent:
        _, err := p.db.Exec(ctx, `
            INSERT INTO nodes (id, org_id, title, content, source, created_at)
            VALUES ($1, $2, $3, $4, $5, $6)`,
            event.StreamID, event.OrgID, evt.Title, evt.Content, evt.Source, event.CreatedAt,
        )
        return err
    case NodeClassifiedEvent:
        _, err := p.db.Exec(ctx,
            `UPDATE nodes SET para = $1 WHERE id = $2`,
            evt.Para, event.StreamID,
        )
        return err
    // ...
    }
    return nil
}
```

---

## Projection Rebuilder

Any projection can be wiped and rebuilt by replaying all events.
This is how you safely evolve schema, fix a projector bug, or add a new read model.

```go
// internal/eventsource/rebuilder.go

type ProjectionRebuilder struct {
    store      EventStore
    projectors []Projector
    logger     *slog.Logger
}

func (r *ProjectionRebuilder) Rebuild(ctx context.Context, streamType string) error {
    r.logger.Info("rebuilding projection", "stream_type", streamType)

    // 1. Truncate the read model (in a transaction)
    // 2. Stream all events of this type from the event store in batches
    // 3. Run each event through relevant projectors
    // 4. Log progress every 1000 events

    return r.streamAndProject(ctx, EventFilter{StreamType: streamType})
}
```

```makefile
# Rebuild a specific projection (safe — does not touch event store)
.PHONY: rebuild-nodes
rebuild-nodes:
	go run ./cmd/admin rebuild-projection --type=node
```

---

## Temporal Queries — The Time Machine

```go
// What did the node look like on a specific date?
node, err := repo.LoadAt(ctx, nodeID, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

// What was in the knowledge graph a month ago?
// Query domain_events WHERE org_id = $1 AND created_at <= $2
// Re-project into an in-memory graph
```

### API endpoint

```
GET /api/v1/nodes/:id?at=2026-01-15T00:00:00Z
```

Returns the node as it existed at that timestamp. Uses `LoadAt` — replays events up to the
given time. This is zero extra storage: the event log already has everything.

---

## Event Schema Evolution

Events are immutable once written. When the schema changes, use **upcasting**:

```go
// internal/eventsource/upcaster.go

type Upcaster interface {
    // Upcast converts an old event schema to the current schema.
    // Called transparently by the EventStore on Load.
    Upcast(raw json.RawMessage, fromVersion int) (DomainEvent, error)
}

// Example: NodeCreatedEvent v1 had no 'source' field. v2 adds it.
func (u *NodeCreatedUpcaster) Upcast(raw json.RawMessage, from int) (DomainEvent, error) {
    switch from {
    case 1:
        var v1 NodeCreatedEventV1
        json.Unmarshal(raw, &v1)
        return NodeCreatedEvent{Title: v1.Title, Content: v1.Content, Source: "unknown"}, nil
    }
    return nil, fmt.Errorf("unknown schema version %d", from)
}
```

Old events are never modified. The upcaster translates on read.

---

## How This Changes Other Features

| Feature | Without ES | With ES |
|---|---|---|
| T-028 Knowledge graph | Source of truth in `nodes` table | `nodes` is a projection; events are truth |
| T-104 HIPAA audit log | Separate `audit_log` table to maintain | `domain_events` IS the audit log |
| T-036 Relevance decay | Track stage in `nodes.review_stage` | `node.reviewed` events; stage is projected |
| T-029 Insight linker | Detects state; creates edges | Subscribes to `ai.insight.generated` events |
| T-016 AI session memory | Separate table | `ai.memory.stored` events; projected |
| T-025 Daily digest | Queries read models | Queries events for the day directly |
| T-048 Chunk planner | Timer state in DB | `chunk.started` / `chunk.completed` events |

---

## Folder Structure

```
internal/eventsource/
├── aggregate.go        — Aggregate base struct
├── event.go            — DomainEvent interface + registry
├── store.go            — EventStore interface
├── store_pg.go         — PostgreSQL implementation
├── projector.go        — Projector interface
├── rebuilder.go        — Projection rebuilder
├── upcaster.go         — Event schema evolution
└── eventsource_test.go — Tests

internal/capture/
├── aggregate.go        — NodeAggregate
├── events.go           — All node event types
├── projector.go        — NodeProjector (sync: nodes table)
├── repository.go       — NodeRepository interface
└── repository_es.go    — EventSourcedNodeRepo
```

---

## Acceptance Criteria

- [ ] `domain_events` table created; `UPDATE`/`DELETE` revoked for app role
- [ ] `EventStore` interface with `Append`, `Load`, `LoadFrom`, `LoadAt`, `Subscribe`
- [ ] `PostgresEventStore` implements all methods
- [ ] Optimistic concurrency: `Append` with wrong `expectedVersion` returns `ErrConcurrencyConflict`
- [ ] `NodeAggregate` implements Create, Tag, Classify, Link, Review, Delete via events
- [ ] `EventSourcedNodeRepo` saves/loads nodes via event replay
- [ ] `NodeProjector` (sync) keeps `nodes` read-model current
- [ ] `EmbeddingProjector` (async via River) generates embeddings after `node.created`
- [ ] `ProjectionRebuilder.Rebuild` replays all events and reconstructs the read model correctly
- [ ] `LoadAt(t)` returns correct node state for any historical timestamp
- [ ] Upcasting: old event schema v1 loaded correctly as v2
- [ ] `GET /api/v1/nodes/:id?at=` returns historical state
- [ ] Concurrent writes to same aggregate: one succeeds, one retries — no data loss
- [ ] 100% test coverage on `internal/eventsource/` (foundational infrastructure)
- [ ] Integration test: create node, tag it, classify it, load at intermediate timestamp — correct state returned
- [ ] Integration test: rebuild projection from events — matches current read model exactly

---

## Dependencies

- T-004 (PostgreSQL — event store is a PostgreSQL table)
- T-115 (Domain events — River-based async projectors use the same event infrastructure)

## Notes

- Start with sync projectors only. Add async (River) projectors for slow operations (embeddings, AI calls).
- Do not try to event-source everything on day one. Start with NodeAggregate. Add TaskAggregate next. CaptureAggregate after.
- The event store is not a message bus. Events are stored durably first, then projected. River reads from the event store — it does not replace it.
- Snapshot optimization: for aggregates with 1000+ events, store a snapshot every N events to avoid full replay on load. Implement this only when benchmarks show it's needed.
- `domain_events` will become the largest table. Partition by `org_id` or `created_at` when it exceeds 100M rows.
