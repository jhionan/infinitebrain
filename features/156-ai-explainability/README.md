# T-156 — Right to Explanation ("Why did IB do this?")

## Overview

EU AI Act Article 86 establishes the right to explanation for AI-driven decisions.
Users can always ask: "Why did IB classify this as a Project? Why is this task priority high?
Why is this insight being surfaced now?"

IB stores a human-readable rationale for every AI decision and exposes it via API.

## AI Explanation Model

```go
// internal/ai/explanation.go

type AIExplanation struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    UserID      uuid.UUID
    NodeID      *uuid.UUID    // the node this decision was made on (if applicable)
    Operation   string        // "classify", "prioritize", "insight", "agent_decompose"
    Decision    string        // what the AI decided: "PARA = project", "priority = high"
    Rationale   string        // human-readable: "Classified as Project because it has a deadline..."
    Confidence  float64
    Factors     []Factor      // machine-readable contributing factors
    Model       string
    CreatedAt   time.Time
}

type Factor struct {
    Name        string   // "has_deadline", "contains_action_verbs", "similar_to_known_project"
    Weight      float64  // how much this contributed (0.0–1.0)
    Value       string   // the actual value: "true", "2026-06-01", "Payments API project"
}
```

## Example Explanations

**Classification:**
```
Operation: classify
Decision: PARA = "project"
Rationale: "This note was classified as a Project because it contains a specific deadline
  (June 2026), references a deliverable ('ship the API'), and uses action-oriented language.
  It shares semantic similarity (0.89) with your existing 'Payments API' project node."
Factors:
  - has_deadline: 0.35 weight, value: "2026-06-01"
  - action_language: 0.28 weight, value: "ship, implement, deliver"
  - semantic_similarity: 0.37 weight, value: "Payments API (0.89)"
```

**Task prioritization:**
```
Operation: prioritize
Decision: priority = "now"
Rationale: "This task is in the 'Now' bucket because: it's blocking 2 other tasks,
  its deadline is in 3 days, and it's aligned with your True North goal of shipping
  by end of Q2 (alignment score: 0.92)."
```

**Agent task decomposition:**
```
Operation: agent_decompose
Decision: "Split into 4 subtasks"
Rationale: "Decomposed into 4 tasks because: 3 business rules apply (JWT expiry, argon2id
  requirement, audit logging), and acceptance criteria for user authentication span 4
  independent verifiable behaviors. Dependencies: T-A002 requires T-A001 because the
  middleware depends on the issuer."
```

## API

```
GET /api/v1/nodes/:id/ai-explanation    — explanation for the latest AI decision on this node
GET /api/v1/ai-explanations/:id         — get a specific explanation by ID
GET /api/v1/ai-explanations?node_id=    — all explanations for a node (history)
```

## Retention

Explanations are retained as long as the node exists + 90 days after deletion.
Users can request their full explanation history (GDPR subject access request support).

## Acceptance Criteria

- [ ] `AIExplanation` entity + table
- [ ] Every AI operation that makes a decision generates an explanation
- [ ] Explanation stored and linked to the node via `metadata.ai.rationale_id`
- [ ] `GET /api/v1/nodes/:id/ai-explanation` returns latest explanation
- [ ] Factors include contributing signals with weights
- [ ] Human-readable `rationale` string always present
- [ ] 90% test coverage

## Dependencies

- T-155 (AI transparency — provenance links to explanation)
- T-154 (EU AI Act — implements Article 86)
- T-021, T-022, T-034, T-151 — all operations that make AI decisions
