# T-128 — Semantic De-duplication on Capture

## Overview

Before creating a new node, check cosine similarity against recent nodes. If a near-duplicate
exists, surface it to the user: link, merge, or create separately. Prevents the knowledge
graph from filling with redundant entries.

## Threshold

`similarity > 0.92` — very high bar. Only flags content that is almost identical in meaning.
Not a soft threshold: false positives are worse than missed duplicates (annoying UX).

## Flow

```
Capture received
    │
    ├── Embed the incoming content
    │
    ├── Query: SELECT id, title, similarity FROM nodes
    │          WHERE org_id = $1
    │            AND 1 - (embedding <=> $2) > 0.92
    │          ORDER BY embedding <=> $2
    │          LIMIT 3
    │
    ├── No matches → create node normally
    │
    └── Match found → emit DuplicateCandidateEvent
            │
            └── API response includes: { "duplicate_of": [{ id, title, similarity }] }
                User decides: link | merge | create_anyway
```

## Merge Logic

```go
// internal/capture/dedup.go

type DedupAction string

const (
    DedupLink        DedupAction = "link"         // create node + add edge to existing
    DedupMerge       DedupAction = "merge"        // append content to existing, discard new
    DedupCreateAnyway DedupAction = "create_anyway" // user knows it's different
)

func (s *CaptureService) ResolveDuplicate(ctx context.Context, req ResolveDuplicateRequest) error {
    switch req.Action {
    case DedupLink:
        // Create new node + relates edge to existing
    case DedupMerge:
        // Append incoming content to existing node content
        // Emit node.content_updated event (T-120)
    case DedupCreateAnyway:
        // Create normally, store dedup_dismissed = true to avoid re-flagging
    }
}
```

## Acceptance Criteria

- [ ] Similarity check runs after embedding, before node creation
- [ ] Candidates returned in capture response when similarity > 0.92
- [ ] `ResolveDuplicate` handles link/merge/create_anyway
- [ ] Merge appends content and emits `node.content_updated` event
- [ ] `dedup_dismissed` flag prevents re-flagging the same pair
- [ ] Performance: check adds < 20ms (HNSW index makes this fast)
- [ ] 90% test coverage

## Dependencies

- T-023 (Semantic search — pgvector)
- T-120 (Event sourcing — merge emits update event)
- T-010 (Notes CRUD — capture pipeline)
