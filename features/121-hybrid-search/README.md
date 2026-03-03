# T-121 — Hybrid Search (BM25 + Vector)

## Overview

Combine PostgreSQL full-text search (BM25 ranking via `ts_rank`) with pgvector cosine
similarity using Reciprocal Rank Fusion (RRF). Consistently outperforms either method alone —
exact keyword matches score high in BM25, semantically related content scores high in vector.

## Why Not Vector-Only

| Query | Vector | BM25 | Hybrid |
|---|---|---|---|
| "OAuth flow" (exact term) | Medium — depends on embedding | High | High |
| "what I learned about auth" | High | Low | High |
| "argon2id" (specific tech term) | Low — rare in training data | High | High |
| "feeling overwhelmed at work" | High | Low | High |

## Implementation

```go
// internal/search/hybrid.go

type HybridSearcher struct {
    db     *pgxpool.Pool
    ai     ai.Provider
    alpha  float64 // weight for BM25 (default 0.5); 1-alpha = vector weight
}

func (s *HybridSearcher) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    embedding, err := s.ai.Embed(ctx, req.Query)
    if err != nil {
        return nil, fmt.Errorf("embedding query: %w", err)
    }

    // Both queries run concurrently
    bm25Results, vectorResults, err := s.runBothQueries(ctx, req, embedding)

    return reciprocalRankFusion(bm25Results, vectorResults, s.alpha), nil
}
```

```sql
-- BM25 query (PostgreSQL full-text search)
SELECT id, ts_rank(search_vector, query) AS score
FROM nodes, plainto_tsquery('english', $1) query
WHERE org_id = $2
  AND search_vector @@ query
ORDER BY score DESC
LIMIT $3;

-- Vector query (pgvector cosine similarity)
SELECT id, 1 - (embedding <=> $1::vector) AS score
FROM nodes
WHERE org_id = $2
  AND embedding IS NOT NULL
ORDER BY embedding <=> $1::vector
LIMIT $3;
```

## Reciprocal Rank Fusion

```go
// RRF score = Σ 1 / (k + rank_i) where k=60 (standard constant)
func reciprocalRankFusion(bm25, vector []RankedResult, alpha float64) []SearchResult {
    scores := map[uuid.UUID]float64{}
    const k = 60

    for rank, r := range bm25 {
        scores[r.ID] += alpha * (1.0 / float64(k+rank+1))
    }
    for rank, r := range vector {
        scores[r.ID] += (1 - alpha) * (1.0 / float64(k+rank+1))
    }
    return sortByScore(scores)
}
```

## Schema Addition

```sql
ALTER TABLE nodes ADD COLUMN search_vector TSVECTOR
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
    ) STORED;

CREATE INDEX ON nodes USING GIN (search_vector);
```

## Acceptance Criteria

- [ ] `HybridSearcher` runs BM25 + vector queries concurrently
- [ ] RRF fusion with configurable alpha weight
- [ ] `search_vector` generated column with GIN index
- [ ] `GET /api/v1/search?q=&mode=hybrid|bm25|vector` — mode selectable per request
- [ ] Benchmark: hybrid outperforms vector-only on a mixed query set (unit test with fixtures)
- [ ] 90% test coverage

## Dependencies

- T-023 (Semantic search — vector infrastructure)
- T-028 (Knowledge graph — nodes table)
