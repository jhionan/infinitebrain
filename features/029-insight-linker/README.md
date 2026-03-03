# T-029 — Cross-Project Insight Linker

## Overview

A nightly background worker that mines the knowledge graph for meaningful connections between nodes that live in different projects. When it finds a high-confidence match — a solution in Project A that addresses a problem in Project B, or a book that relates to a technical decision — it creates an `insight` node and surfaces it in the daily digest.

The brain gets smarter over time. Connections that would never occur to you manually get surfaced automatically.

---

## Why

A software engineer maintains mental context across multiple projects. Business rules, patterns, and solutions recur — but in a brain that holds 10 projects, the connections are invisible. The insight linker makes them visible.

This is also the mechanism that turns Infinite Brain from a storage system into an **active collaborator**.

---

## How It Works

```
Nightly cron (03:00 local time)
    └── For each user with nodes in 2+ projects:
            1. Fetch all nodes with embeddings
            2. Compute pairwise cosine similarity for cross-project pairs
            3. Filter: similarity > threshold AND different project_id AND no edge exists
            4. For top candidates: ask AI to validate and describe the connection
            5. Create insight node + edges
            6. Queue for next daily digest
```

### Similarity Threshold

Start at `0.82`. Below this, matches are too generic to be useful (e.g., two nodes both about "authentication" in different projects — obvious, not insightful). Tunable per user via config.

---

## Schema

No new tables. Uses the existing `nodes` and `edges` tables (T-028).

```sql
-- Insight node example
INSERT INTO nodes (user_id, type, title, content, para, metadata) VALUES (
  $user_id,
  'insight',
  'Pattern connection: Project A → Project B',
  'The guard clause pattern you used in validateTransfer() in Project A directly solves
   the input validation problem you noted in Project B (note: "how to handle invalid webhook payloads")',
  NULL,  -- insights span PARA categories
  '{"confidence": 0.91, "source": "insight-linker", "from_project": "...", "to_project": "..."}'
);

-- Edges: insight relates to both source nodes
INSERT INTO edges (user_id, from_node_id, to_node_id, relation_type, confidence, created_by)
VALUES ($user_id, $insight_id, $node_a_id, 'relates', 0.91, 'insight-linker'),
       ($user_id, $insight_id, $node_b_id, 'solves',  0.91, 'insight-linker');
```

---

## Asynq Job

```go
// Job name
const JobInsightLinker = "ai:insight_linker"

// Payload
type InsightLinkerPayload struct {
    UserID uuid.UUID `json:"user_id"`
}

// Registered as a cron job: runs nightly at 03:00
// asynq.NewTask(JobInsightLinker, payload, asynq.Unique(24*time.Hour))
```

---

## AI Validation Step

Raw cosine similarity catches candidates. AI validates them to avoid noise.

Prompt (simplified):
```
Node A (Project: Payments API):
"Guard clause pattern: validateTransfer() returns early if amount > limit"

Node B (Project: Webhook Service):
"Problem: need to reject malformed payloads without panicking"

Are these meaningfully connected? If yes, describe the connection in one sentence.
Respond: { "connected": true/false, "description": "...", "relation": "solves|relates|inspired_by" }
```

If AI says `connected: false` → discard candidate, no node or edge created.

---

## Deduplication

Before creating an edge, check:
```sql
SELECT 1 FROM edges
WHERE from_node_id = $a AND to_node_id = $b AND relation_type = $rel
UNION
SELECT 1 FROM edges
WHERE from_node_id = $b AND to_node_id = $a AND relation_type = $rel
```

If exists → skip. The unique constraint on `edges` also enforces this at DB level.

---

## Daily Digest Integration

Insights created by the linker are included in the next daily digest (T-025) under a dedicated section:

```
--- New Connections Found ---

Your guard clause pattern in Payments API may solve the payload
validation problem you noted in Webhook Service.
[View connection] [Dismiss]
```

User can dismiss an insight → edge gets `dismissed_at` timestamp, excluded from future digests.

---

## Performance

- Users with few nodes: full pairwise scan is fine (< 100ms)
- Users with 1000+ nodes: use pgvector ANN (approximate nearest neighbor) per node, filter cross-project pairs from results
- Maximum candidates per run: 50 (configurable). AI validation is the expensive step — cap it.
- Job is idempotent: `asynq.Unique(24*time.Hour)` prevents double-runs

---

## Acceptance Criteria

- [ ] Nightly cron job registered in Asynq scheduler
- [ ] Job fetches all nodes with embeddings for a user, grouped by project
- [ ] Cosine similarity computed for cross-project pairs (pgvector `<=>` operator)
- [ ] Candidates above threshold passed to AI validation
- [ ] Validated insights create an `insight` node + 2 edges
- [ ] Duplicate edges not created (check before insert)
- [ ] Dismissed insights not re-surfaced
- [ ] Insights appear in daily digest payload
- [ ] Job is idempotent (safe to run multiple times)
- [ ] Unit tests for similarity filtering and deduplication logic
- [ ] Integration test: two projects with known-similar nodes produce an insight
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL + pgvector)
- T-020 (AI provider — validation prompt)
- T-023 (Semantic search — pgvector ANN)
- T-025 (Daily digest — surfacing insights)
- T-028 (Knowledge graph — nodes and edges schema)

## Notes

- Run at 03:00 to avoid overlap with peak usage and digest generation (T-025 runs at 07:00)
- Users with only one project skip the job entirely
- First run after a new project is created may produce many candidates — cap at 20 insights per first run per new project
