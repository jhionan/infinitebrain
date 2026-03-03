# T-131 — Business Rule Nodes + Conflict Detection

## Overview

Business rules are first-class citizens in the knowledge graph. They are not just notes — they
are structured, versioned, and machine-queryable constraints that govern how the system (and any
AI agent working in the system) must behave.

Every requirement, constraint, and policy is a `BusinessRule` node. When agents execute tasks,
they receive the relevant rules as part of their context. When rules conflict, IB detects it.

## Why This Matters

Without structured rules:
- AI agents hallucinate or apply wrong assumptions
- Conflicting decisions accumulate silently across the codebase
- Onboarding is slow — the "why" behind decisions lives in people's heads

With structured rules:
- Every agent task includes "rules that apply to this task"
- New rules are checked against existing rules for conflicts before acceptance
- The full rule set is queryable: "what are all authentication rules?"

## BusinessRule Node

```go
// internal/domain/business_rule.go

type BusinessRule struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    Name        string       // e.g. "passwords must use argon2id"
    Category    RuleCategory // security, auth, data, performance, compliance, product
    Description string       // full human-readable rule
    Rationale   string       // WHY this rule exists
    Source      string       // "HIPAA §164.312", "ADR-003", "legal"
    Severity    RuleSeverity // must, should, may (RFC 2119)
    Tags        []string
    ConflictsWith []uuid.UUID // IDs of rules this potentially conflicts with
    Version     int
    Active      bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type RuleCategory string
const (
    RuleSecurity    RuleCategory = "security"
    RuleAuth        RuleCategory = "auth"
    RuleData        RuleCategory = "data"
    RuleCompliance  RuleCategory = "compliance"
    RuleProduct     RuleCategory = "product"
    RuleArchitecture RuleCategory = "architecture"
)

type RuleSeverity string
const (
    RuleMust   RuleSeverity = "must"   // non-negotiable
    RuleShould RuleSeverity = "should" // strong recommendation
    RuleMay    RuleSeverity = "may"    // optional / context-dependent
)
```

## Conflict Detection

```go
// internal/business_rules/conflict.go

type ConflictDetector struct {
    store RuleStore
    ai    Provider
}

// DetectConflicts checks a new/updated rule against all active rules in the same org.
// Uses semantic similarity to find potentially conflicting rules, then asks AI to judge.
func (d *ConflictDetector) DetectConflicts(ctx context.Context, candidate BusinessRule) ([]Conflict, error) {
    // 1. Semantic search: find rules with high similarity to candidate
    similar, err := d.store.FindSimilar(ctx, candidate.OrgID, candidate.Description, 0.75)

    // 2. For each similar rule, ask AI: do these conflict?
    var conflicts []Conflict
    for _, existing := range similar {
        result, err := d.ai.Complete(ctx, conflictPrompt(candidate, existing))
        if result.Conflicts {
            conflicts = append(conflicts, Conflict{
                RuleA:       candidate.ID,
                RuleB:       existing.ID,
                Explanation: result.Explanation,
            })
        }
    }
    return conflicts, nil
}
```

## API

```
POST   /api/v1/rules              — create rule
GET    /api/v1/rules              — list rules (filter by category, severity, tags)
GET    /api/v1/rules/:id          — get rule with version history
PUT    /api/v1/rules/:id          — update rule (creates new version via event)
DELETE /api/v1/rules/:id          — deactivate rule

GET    /api/v1/rules/search?q=    — semantic search over rules
POST   /api/v1/rules/check-conflict — check a draft rule for conflicts before saving
```

## Integration with Agent Tasks (T-134)

When IB creates an `AgentTask`, it:
1. Determines the task category (auth, data handling, API design, etc.)
2. Fetches all `must` and `should` rules for that category
3. Injects them into the agent's system prompt: "You must follow these rules..."
4. The agent cannot proceed without acknowledging the rules

## Event Sourcing (T-120)

BusinessRule changes are domain events:
- `rule.created` — new rule added
- `rule.updated` — rule text or severity changed (version bump)
- `rule.deactivated` — rule retired
- `rule.conflict_detected` — potential conflict found with another rule

## Acceptance Criteria

- [ ] `BusinessRule` entity with all fields above
- [ ] CRUD endpoints for rules
- [ ] Semantic search over rule descriptions
- [ ] Conflict detection runs automatically on rule create/update
- [ ] Conflicts surface as warnings (non-blocking) — user resolves
- [ ] Rules linked to their source (ADR ID, compliance doc, etc.)
- [ ] Version history via event sourcing
- [ ] Rules injected into agent context (T-134 integration)
- [ ] 90% test coverage

## Dependencies

- T-028 (knowledge graph — rules are nodes in the graph)
- T-120 (event sourcing — rule changes are events)
- T-023 (semantic search — similarity-based conflict detection)
- T-134 (agent tasks — rules injected as context)
