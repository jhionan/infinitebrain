# T-124 — AI Cost Attribution

## Overview

Track every AI API call with token counts, latency, and real dollar cost. Aggregate per
user per day. Surface in the billing API and compliance dashboard. Enables the metered
billing model and shows users exactly what they're spending.

## Schema

```sql
CREATE TABLE ai_usage (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id),
    org_id        UUID NOT NULL REFERENCES organizations(id),
    provider      TEXT NOT NULL,      -- 'anthropic' | 'openai'
    model         TEXT NOT NULL,      -- 'claude-sonnet-4-6' | 'text-embedding-3-small'
    operation     TEXT NOT NULL,      -- 'classify' | 'tag' | 'embed' | 'qa' | 'digest' | 'compress'
    input_tokens  INT NOT NULL,
    output_tokens INT NOT NULL,
    cost_usd      NUMERIC(10, 8) NOT NULL,
    latency_ms    INT NOT NULL,
    trace_id      TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON ai_usage (org_id, created_at DESC);
CREATE INDEX ON ai_usage (user_id, created_at DESC);

-- Daily aggregates (materialized, refreshed hourly)
CREATE MATERIALIZED VIEW ai_usage_daily AS
SELECT
    user_id,
    org_id,
    date_trunc('day', created_at) AS day,
    operation,
    SUM(input_tokens + output_tokens) AS total_tokens,
    SUM(cost_usd) AS total_cost_usd,
    COUNT(*) AS call_count,
    AVG(latency_ms) AS avg_latency_ms
FROM ai_usage
GROUP BY user_id, org_id, date_trunc('day', created_at), operation;

CREATE UNIQUE INDEX ON ai_usage_daily (user_id, day, operation);
```

## Cost Tracking Middleware

```go
// internal/ai/cost_middleware.go

type CostTrackingProvider struct {
    inner    Provider
    recorder UsageRecorder
    pricing  PricingTable   // loaded from config; updated when models change
}

func (p *CostTrackingProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    start := time.Now()
    resp, err := p.inner.Complete(ctx, req)
    if err != nil {
        return resp, err
    }

    cost := p.pricing.Calculate(req.Model, resp.InputTokens, resp.OutputTokens)
    p.recorder.Record(ctx, AIUsage{
        UserID:       auth.UserIDFromContext(ctx),
        OrgID:        auth.OrgIDFromContext(ctx),
        Provider:     req.Provider,
        Model:        req.Model,
        Operation:    req.Operation,
        InputTokens:  resp.InputTokens,
        OutputTokens: resp.OutputTokens,
        CostUSD:      cost,
        LatencyMS:    int(time.Since(start).Milliseconds()),
        TraceID:      trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
    })
    return resp, nil
}
```

## Pricing Table

```go
// pkg/ai/pricing.go — updated when providers change prices

var DefaultPricing = PricingTable{
    "claude-sonnet-4-6": {InputPer1M: 3.00, OutputPer1M: 15.00},
    "claude-haiku-4-5":  {InputPer1M: 0.25, OutputPer1M: 1.25},
    "text-embedding-3-small": {InputPer1M: 0.02, OutputPer1M: 0},
    "whisper-1":         {PerMinute: 0.006},
}
```

## Threshold Alerts

```go
// Daily cost threshold check — River job every hour
if dailyCost > user.DailyCostThresholdUSD {
    events.Emit(ctx, AICostThresholdEvent{UserID, dailyCost, threshold})
    // → notification to user
    // → throttle AI features if above hard limit
}
```

## API

```
GET /api/v1/usage/ai?from=&to=&group_by=operation|day|model
GET /api/v1/admin/usage/ai?org_id=&from=&to=  (admin only)
```

## Acceptance Criteria

- [ ] `CostTrackingProvider` wraps any `Provider` transparently
- [ ] All AI calls (Complete, Embed, Transcribe) record to `ai_usage`
- [ ] Pricing table configurable via env/config without code change
- [ ] Materialized view refreshed hourly via River cron
- [ ] Threshold alert fires when daily cost exceeds user limit
- [ ] Usage API returns breakdown by operation, model, day
- [ ] 90% test coverage

## Dependencies

- T-020 (AI provider — wraps Provider interface)
- T-120 (Event sourcing — `ai.cost.threshold` event)
