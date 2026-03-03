# T-132 — Requirements + Acceptance Criteria Triad

## Overview

Requirements are structured nodes that link to machine-verifiable acceptance criteria.
When a requirement is added to IB, it can generate test stubs automatically (T-133).
When an agent implements a task derived from a requirement, the pre-generated tests verify completion.

This is TDD at the product level: the acceptance criteria are written before implementation begins.

## The Triad

```
Requirement
    └── AcceptanceCriteria (1..n)
            └── GeneratedTest (0..n)  ← T-133 populates this
```

## Data Model

```go
// internal/domain/requirement.go

type Requirement struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    Title       string
    Description string       // full requirement text
    Category    string       // functional, non-functional, security, performance
    Priority    Priority     // must_have, should_have, nice_to_have
    Source      string       // "user story", "ADR-003", "regulatory"
    LinkedRules []uuid.UUID  // business rules that apply to this requirement
    Criteria    []AcceptanceCriteria
    Status      RequirementStatus // draft, active, implemented, deprecated
    Tags        []string
}

type AcceptanceCriteria struct {
    ID            uuid.UUID
    RequirementID uuid.UUID
    Description   string  // "Given X, when Y, then Z" format
    Verifiable    bool    // can this be machine-checked?
    TestStatus    TestStatus // not_generated, generated, passing, failing
    GeneratedTest *GeneratedTest  // set after T-133 runs
}

type GeneratedTest struct {
    ID            uuid.UUID
    CriteriaID    uuid.UUID
    Language      string   // "go"
    TestCode      string   // the generated test stub
    LastRunAt     *time.Time
    LastResult    *TestResult
}
```

## Acceptance Criteria Format

IB uses Given/When/Then format for criteria — this is the most AI-parseable format
for test generation:

```
Given: a user with valid credentials
When: they submit a login request
Then: they receive a JWT valid for 30 days
 And: the response includes a refresh token
 And: the event auth.login_success is emitted
```

Each "Then" / "And" clause maps to one assertion in the generated test.

## API

```
POST   /api/v1/requirements                    — create requirement
GET    /api/v1/requirements                    — list (filter by category, priority, status)
GET    /api/v1/requirements/:id                — get with criteria and test status
PUT    /api/v1/requirements/:id                — update
POST   /api/v1/requirements/:id/criteria       — add acceptance criterion
DELETE /api/v1/requirements/:id/criteria/:cid  — remove criterion

POST   /api/v1/requirements/:id/generate-tests — trigger T-133 test generation
GET    /api/v1/requirements/:id/test-status    — coverage status across all criteria
```

## Requirement Coverage Report

```go
// GET /api/v1/requirements/coverage

type CoverageReport struct {
    Total         int
    Implemented   int     // requirements with all criteria passing
    Partial        int     // some criteria passing
    NotStarted    int     // no tests generated
    CoverageRatio float64 // implemented / total
    Gaps          []RequirementGap
}

type RequirementGap struct {
    RequirementID uuid.UUID
    Title         string
    MissingCriteria []string
}
```

This is the foundation for T-136 (gap analysis) — IB can report on which requirements
are not implemented.

## Acceptance Criteria (for this task)

- [ ] `Requirement` entity with criteria sub-entities
- [ ] Given/When/Then criteria format validated on input
- [ ] CRUD for requirements and criteria
- [ ] Coverage report endpoint
- [ ] Requirements link to business rules (T-131)
- [ ] Test status tracked per criterion
- [ ] T-133 integration: generate-tests endpoint triggers test generation
- [ ] 90% test coverage

## Dependencies

- T-028 (knowledge graph — requirements are nodes)
- T-120 (event sourcing — requirement changes are events)
- T-131 (business rules — requirements reference applicable rules)
- T-133 (test generation — auto-generates tests from criteria)
