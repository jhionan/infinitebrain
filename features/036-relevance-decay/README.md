# T-036 — Relevance Decay + Review Ladder

## Overview

Every node in the knowledge graph has a relevance lifecycle. Items that are never re-confirmed slowly progress through a review ladder and are eventually deleted. Items the user confirms as still relevant are checked every 6 months. The system ensures the brain stays current — a movie on a watchlist for 10 years is not relevant anymore.

Silence never advances the ladder. Only an explicit "No" does.

---

## Why

Every second brain becomes a graveyard. Notes accumulate, watchlists grow, tasks pile up — and nothing gets cleaned out because there's no forcing function. This feature is the forcing function. It is built into the data model, not bolted on as a manual review feature.

---

## Review Ladder

| Stage | Label | Check Interval | "Yes" | "No" | Silence |
|---|---|---|---|---|---|
| 0 | pending | bi-weekly (14 days) | → confirmed (1) | → doubt_1 (2) | resurface same interval |
| 1 | confirmed | 6 months | stay at 1 | → doubt_1 (2) | resurface same interval |
| 2 | doubt_1 | 1 month | → confirmed (1) | → doubt_2 (3) | resurface same interval |
| 3 | doubt_2 | 3 months | → confirmed (1) | → doubt_3 (4) | resurface same interval |
| 4 | doubt_3 | 6 months | → confirmed (1) | → doubt_4 (5) | resurface same interval |
| 5 | doubt_4 | 6 months | → confirmed (1) | **hard delete** | resurface same interval |

**Rules:**
- Silence = no state change, item resurfaces again at the same interval on the next cron run
- "Yes" at any stage = jump to stage 1 (confirmed), `next_review_at = now() + 6 months`
- "No" at stage 5 = hard delete (no recovery)
- Total time from first "No" to deletion: 1 + 3 + 6 + 6 = **16 months**

---

## Schema

Two columns added to `nodes` table (T-028):

```sql
ALTER TABLE nodes ADD COLUMN review_stage    SMALLINT     NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN next_review_at  TIMESTAMPTZ  DEFAULT (now() + interval '14 days');

-- Indexes for efficient cron queries
CREATE INDEX ON nodes (user_id, review_stage, next_review_at)
    WHERE deleted_at IS NULL AND archived_at IS NULL;
```

Stage constants in Go:
```go
const (
    ReviewStagePending   = 0 // bi-weekly
    ReviewStageConfirmed = 1 // 6 months
    ReviewStageDoubt1    = 2 // 1 month
    ReviewStageDoubt2    = 3 // 3 months
    ReviewStageDoubt3    = 4 // 6 months
    ReviewStageDoubt4    = 5 // 6 months → delete on No
)

var stageIntervals = map[int]time.Duration{
    ReviewStagePending:   14 * 24 * time.Hour,
    ReviewStageConfirmed: 180 * 24 * time.Hour,
    ReviewStageDoubt1:    30 * 24 * time.Hour,
    ReviewStageDoubt2:    90 * 24 * time.Hour,
    ReviewStageDoubt3:    180 * 24 * time.Hour,
    ReviewStageDoubt4:    180 * 24 * time.Hour,
}
```

---

## State Machine

```go
// internal/adhd/review.go

func Transition(stage int, response ReviewResponse) (newStage int, nextInterval time.Duration, delete bool) {
    switch response {
    case ReviewYes:
        return ReviewStageConfirmed, stageIntervals[ReviewStageConfirmed], false
    case ReviewNo:
        if stage == ReviewStageDoubt4 {
            return 0, 0, true // delete
        }
        next := nextStage(stage)
        return next, stageIntervals[next], false
    case ReviewSilence:
        return stage, stageIntervals[stage], false // no change
    }
}

func nextStage(stage int) int {
    switch stage {
    case ReviewStagePending:   return ReviewStageDoubt1
    case ReviewStageConfirmed: return ReviewStageDoubt1
    case ReviewStageDoubt1:    return ReviewStageDoubt2
    case ReviewStageDoubt2:    return ReviewStageDoubt3
    case ReviewStageDoubt3:    return ReviewStageDoubt4
    default:                   return stage
    }
}
```

---

## Cron Worker

Runs nightly. Batches items due for review into the daily digest.

```go
// Job: ai:review_queue
// Schedule: daily at 07:00 (same run as daily digest T-025)

func (w *ReviewWorker) Run(ctx context.Context) error {
    nodes, err := w.repo.FetchDueForReview(ctx, userID, time.Now())
    // FetchDueForReview: WHERE next_review_at <= now() AND deleted_at IS NULL
    // Limit: 10 items per digest to avoid overwhelming the user
    ...
    // Append to digest payload
}
```

---

## Review Response API

```
POST /api/v1/nodes/:id/review
Body: { "response": "yes" | "no" }
```

Handler calls `Transition()`, writes new `review_stage` and `next_review_at`. If `delete = true`, hard-deletes the node and all its edges.

No batch endpoint at MVP — user responds per item from the digest.

---

## Digest Integration

Review items appear in the daily digest (T-025) under a section:

```
--- Worth keeping? ---

[ ] Watch Dune 2         (saved 14 months ago)  [Keep] [Let go]
[ ] Read Atomic Habits   (saved 2 years ago)    [Keep] [Let go]
[ ] Call dentist         (saved 8 months ago)   [Keep] [Let go]
```

Max 10 items per digest. Items are sorted by `review_stage DESC` — items closest to deletion shown first.

---

## Node Types Exempt from Decay

Some nodes should never enter the review ladder:

| Type | Reason |
|---|---|
| `project` | Managed by project status, not decay |
| `insight` | AI-generated, curated separately |
| `event` with future `scheduled_at` | Not yet relevant to review |

Set `next_review_at = NULL` for exempt nodes. Cron query filters `WHERE next_review_at IS NOT NULL`.

Events with a past `scheduled_at` automatically advance to stage 2 (doubt_1) — the appointment is done, was it useful to capture?

---

## Acceptance Criteria

- [ ] `review_stage` and `next_review_at` columns added to `nodes` via migration
- [ ] `Transition()` function covers all stage/response combinations
- [ ] POST /api/v1/nodes/:id/review applies transition and persists
- [ ] "No" at stage 5 hard-deletes node and cascades edges
- [ ] Cron worker fetches due nodes, max 10 per user per run
- [ ] Items appear in daily digest payload
- [ ] Exempt node types skip the review ladder (`next_review_at = NULL`)
- [ ] Past-date events automatically move to doubt_1
- [ ] Unit tests for `Transition()` covering all 18 state combinations (6 stages × 3 responses)
- [ ] Integration test: full ladder progression from stage 0 to deletion
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL)
- T-025 (Daily digest — surfacing review items)
- T-028 (Knowledge graph — nodes table)

## Notes

- Review responses from Telegram/WhatsApp bots should work: "keep" / "let go" as bot commands
- Future: AI auto-confirmation for nodes with very high connectivity (many edges) — highly linked nodes are probably still relevant. Not MVP.
