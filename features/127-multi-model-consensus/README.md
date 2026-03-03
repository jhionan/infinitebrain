# T-127 — Multi-Model Consensus Classifier

## Overview

For high-stakes or low-confidence classifications, ask 2 independent AI models and take
the majority decision. If they disagree, flag for user review. Statistically more accurate
than single-model classification for edge cases.

## When Consensus Is Used

| Trigger | Example |
|---|---|
| Node type contains PHI indicator | "my antidepressant dose" |
| First classification of a new project | No prior context to guide the model |
| Business rule node | Rules where misclassification has real consequences |
| Confidence below threshold | Primary model returns confidence < 0.7 |
| User has opted into high-accuracy mode | Power user setting |

## Implementation

```go
// internal/ai/consensus.go

type ConsensusClassifier struct {
    models    []Provider   // e.g. [ClaudeProvider, OpenAIProvider]
    threshold float64      // agreement ratio — 1.0 = all must agree, 0.5 = majority
    recorder  UsageRecorder
}

type ConsensusResult struct {
    Decision    ClassificationResult
    Confidence  float64   // how many models agreed
    Agreed      bool      // true if >= threshold fraction agreed
    Responses   []ModelResponse
    FlaggedForReview bool
}

func (c *ConsensusClassifier) Classify(ctx context.Context, content string) (ConsensusResult, error) {
    results := make(chan ModelResponse, len(c.models))

    // All models run concurrently
    var wg sync.WaitGroup
    for _, model := range c.models {
        wg.Add(1)
        go func(m Provider) {
            defer wg.Done()
            resp, err := m.Complete(ctx, classifyRequest(content))
            results <- ModelResponse{Provider: m.Name(), Result: resp, Err: err}
        }(model)
    }
    wg.Wait()
    close(results)

    return aggregate(results, c.threshold), nil
}
```

## Aggregation

```go
func aggregate(responses []ModelResponse, threshold float64) ConsensusResult {
    // Count votes per PARA category
    votes := map[string]int{}
    for _, r := range responses {
        if r.Err == nil {
            votes[r.Result.Para]++
        }
    }

    winner, count := topVote(votes)
    agreementRatio := float64(count) / float64(len(responses))

    return ConsensusResult{
        Decision:         ClassificationResult{Para: winner},
        Confidence:       agreementRatio,
        Agreed:           agreementRatio >= threshold,
        FlaggedForReview: agreementRatio < threshold,
    }
}
```

## Cost Control

Consensus costs 2× per call. Used only on qualifying nodes (not every capture).
Cost is tracked per call via T-124 with `operation = "classify_consensus"`.

## Acceptance Criteria

- [ ] `ConsensusClassifier` runs N models concurrently
- [ ] Agreement ratio computed; below threshold → `FlaggedForReview = true`
- [ ] Flagged nodes surface to user for manual classification
- [ ] PHI nodes always use consensus (never single-model)
- [ ] Cost recorded per consensus call (T-124)
- [ ] Unit tests: 2 models agree → confident result; 2 models disagree → flagged
- [ ] 90% test coverage

## Dependencies

- T-020 (AI provider abstraction)
- T-124 (AI cost attribution)
- T-104 (PHI detection triggers consensus)
