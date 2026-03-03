# T-122 — Prompt Versioning + A/B Testing

## Overview

Treat prompts as versioned code. Run A/B experiments on prompt effectiveness. Measure
real outcomes (user corrections, satisfaction) not just LLM output quality. Graduate
winners automatically.

## Prompt Registry

```
internal/ai/prompts/
├── registry.go           — prompt loader + version resolution
├── classify/
│   ├── v1.go             — production (100% of traffic)
│   └── v2.go             — experiment (20% of traffic)
├── tag/
│   └── v1.go
├── digest/
│   └── v1.go
└── qa/
    └── v1.go
```

```go
// internal/ai/prompts/registry.go

type PromptVersion struct {
    Version     int
    Template    string
    Author      string
    Description string   // what changed vs previous version
    CreatedAt   time.Time
}

type Registry struct {
    mu      sync.RWMutex
    prompts map[string][]PromptVersion  // key: "classify", "tag", etc.
}

// Resolve returns the prompt version for this request based on active experiments.
func (r *Registry) Resolve(ctx context.Context, name string, orgID uuid.UUID) *PromptVersion
```

## A/B Experiment Table

```sql
CREATE TABLE prompt_experiments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_name  TEXT NOT NULL,          -- 'classify' | 'tag' | 'qa'
    variant_a    INT NOT NULL,           -- version number
    variant_b    INT NOT NULL,           -- version number
    traffic_b    FLOAT NOT NULL,         -- 0.0–1.0 fraction sent to B
    org_ids      UUID[],                 -- null = all orgs; specific list = targeted
    status       TEXT NOT NULL,          -- 'running' | 'paused' | 'graduated' | 'rolled_back'
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at     TIMESTAMPTZ,
    winner       TEXT                    -- 'a' | 'b' | null (undecided)
);

CREATE TABLE prompt_outcomes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id   UUID REFERENCES prompt_experiments(id),
    variant         TEXT NOT NULL,      -- 'a' | 'b'
    node_id         UUID,               -- the node that was processed
    operation       TEXT NOT NULL,      -- 'classify' | 'tag'
    ai_output       JSONB NOT NULL,     -- what the AI returned
    user_corrected  BOOLEAN,            -- did user change the AI's output?
    correction      JSONB,              -- what they changed it to
    latency_ms      INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Outcome Measurement

```go
// When AI classifies a node, record the outcome
outcome := PromptOutcome{
    ExperimentID: activeExperiment.ID,
    Variant:      resolvedVariant,
    NodeID:       node.ID,
    Operation:    "classify",
    AIOutput:     classificationResult,
    LatencyMS:    latency,
}
outcomeRepo.Record(ctx, outcome)

// When user moves a node to a different PARA category:
// → this triggers an update to outcome: user_corrected = true
// This is the ground truth signal for prompt quality
```

## Auto-graduation

A River cron evaluates experiments weekly:
- If variant B has correction rate < variant A for 30 days with p < 0.05 → graduate B
- If variant B has correction rate > variant A by 10% → roll back automatically

## Acceptance Criteria

- [ ] Prompt registry loads versions from Go constants (no DB for prompt text)
- [ ] Traffic splitting: deterministic per `org_id` (same org always gets same variant)
- [ ] `prompt_experiments` + `prompt_outcomes` tables
- [ ] User correction captured as outcome signal (fires `node.classified` event with correction)
- [ ] Weekly River job evaluates significance and updates experiment status
- [ ] Admin API: `GET /api/v1/admin/experiments` — current experiments + correction rates
- [ ] 90% test coverage on traffic splitting and significance calculation

## Dependencies

- T-120 (Event sourcing — correction events)
- T-115 (Domain events — outcome recording)
