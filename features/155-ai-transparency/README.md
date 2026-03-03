# T-155 — AI Transparency Labeling

## Overview

Every AI-generated output in Infinite Brain must be labeled as such.
This satisfies EU AI Act Article 52(3) and builds user trust: users always know what came
from AI vs. what they wrote themselves.

## Fields Added to AI-Generated Nodes

```go
// Added to Node (via metadata JSONB, or dedicated columns)

type AIProvenance struct {
    Generated  bool      // true if content was AI-generated or AI-modified
    Model      string    // "claude-sonnet-4-6", "whisper-1", etc.
    Operation  string    // "classify", "tag", "summarize", "qa", "digest", "agent"
    Confidence float64   // 0.0–1.0 — model's confidence in the output
    RationaleID *uuid.UUID // FK to ai_explanations table (T-156)
    GeneratedAt time.Time
}
```

Stored in `nodes.metadata`:
```json
{
  "ai": {
    "generated": true,
    "model": "claude-sonnet-4-6",
    "operation": "classify",
    "confidence": 0.91,
    "rationale_id": "...",
    "generated_at": "2026-04-01T10:00:00Z"
  }
}
```

## API Response Labeling

Every response that includes AI-generated content includes provenance:

```json
{
  "id": "...",
  "title": "Meeting notes: Product sync",
  "para": "project",
  "tags": ["meetings", "product"],
  "ai": {
    "generated": true,
    "model": "claude-sonnet-4-6",
    "operation": "classify",
    "confidence": 0.91,
    "explanation_url": "/api/v1/nodes/ID/ai-explanation"
  }
}
```

## Agent Output Labeling

Tasks and content created by AI agents (T-134) carry the full agent provenance:

```json
{
  "ai": {
    "generated": true,
    "model": "claude-sonnet-4-6",
    "operation": "agent_task",
    "agent_task_id": "...",
    "goal_id": "...",
    "confidence": 0.88
  }
}
```

## Acceptance Criteria

- [ ] `AIProvenance` struct + storage in `nodes.metadata.ai`
- [ ] All AI operations (classify, tag, embed, qa, digest, agent) set provenance
- [ ] API responses include `ai` provenance block when content is AI-generated
- [ ] Human-created nodes have `ai.generated = false` (default)
- [ ] User corrections (T-129) mark the node as human-corrected: `ai.corrected_by = "user"`
- [ ] 90% test coverage

## Dependencies

- T-021 (classify), T-022 (tag), T-024 (qa), T-025 (digest) — all set provenance
- T-134 (agent tasks) — agent output carries full provenance
- T-154 (EU AI Act — this implements Article 52(3))
- T-156 (explainability — provenance links to explanation)
