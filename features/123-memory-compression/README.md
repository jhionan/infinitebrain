# T-123 — Memory Compression (Nightly Consolidation)

## Overview

Human memory consolidates during sleep — individual experiences compress into abstractions.
Infinite Brain does the same. A nightly River job identifies clusters of old `agent_memories`
for a project, summarizes them into a single high-density context memory, and expires the
originals.

The AI brain gets smarter over time without growing unbounded.

## How It Works

```
Nightly at 02:00 (before insight linker at 03:00)
    └── For each user with memories older than 30 days:
            1. Cluster memories by project_id + type
            2. For clusters with 10+ memories: trigger compression
            3. AI reads the cluster, writes a summary memory
            4. Original memories marked expires_at = now()
            5. Summary stored as type='context', confidence=0.9
```

## Compression Prompt

```go
// internal/ai/prompts/compress/v1.go

const V1 = `You are compressing a set of AI reasoning memories into a concise summary.

These are memories accumulated while working on: {{.ProjectName}}

Memories to compress:
{{range .Memories}}- [{{.Type}}] {{.Content}}
{{end}}

Write a single dense summary (max 500 words) that captures:
1. Key decisions made and their reasoning
2. Patterns observed about the user's work style
3. Problems encountered and how they were resolved
4. Critical context a future AI session must know

Return JSON: {"summary": "...", "key_facts": ["...", "..."], "confidence": 0.0-1.0}`
```

## Schema

```sql
ALTER TABLE agent_memories ADD COLUMN compressed_from UUID[];
-- IDs of the memories this summary replaces
-- Allows tracing what was compressed into what
```

## River Job

```go
const JobMemoryCompression = "ai:memory_compression"

type MemoryCompressionPayload struct {
    UserID    uuid.UUID
    ProjectID uuid.UUID
    MemoryIDs []uuid.UUID  // the cluster to compress
}
```

## Compression Rules

- Minimum cluster size: 10 memories (no compression for small sets)
- Maximum age before eligible: 30 days (recent memories stay raw)
- Never compress: `type = 'error'` (errors kept for debugging)
- Never compress: `confidence < 0.5` (uncertain memories flagged, not compressed)
- Max 3 compression levels: memories can be compressed once, not recursively

## Acceptance Criteria

- [ ] River cron at 02:00 schedules compression per user
- [ ] Clustering by project_id + type
- [ ] Compression prompt produces structured JSON with summary + key_facts
- [ ] Original memories set `expires_at = now()` after compression
- [ ] Summary memory stores `compressed_from` array
- [ ] `context` memory with compression result stored and embedded
- [ ] Memory count per project visibly drops over time (integration test)
- [ ] 90% test coverage

## Dependencies

- T-016 (AI session memory — agent_memories table)
- T-120 (Event sourcing — `ai.memory.compressed` event)
- T-020 (AI provider — compression prompt call)
