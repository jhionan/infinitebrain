# T-134 — AgentTask Entity + Agent Dispatch Loop

## Overview

IB can decompose a high-level goal into concrete, verified `AgentTask` nodes and dispatch
AI agents to execute them. Each agent receives full domain context from the knowledge base —
relevant business rules, requirements, ADRs, and acceptance tests.

When an agent completes a task, the pre-generated tests verify the result. Completion is
proven, not assumed.

This is the engine that makes IB a project manager — not just a note-taking tool.

## The Loop

```
User: "implement JWT authentication per our security spec"
    │
    ├── IB decomposes into AgentTasks:
    │   ├── T-A001: Create JWT issuer service (blocked by: none)
    │   ├── T-A002: Create JWT validation middleware (blocked by: T-A001)
    │   ├── T-A003: Integrate with user service (blocked by: T-A001, T-A002)
    │   └── T-A004: Write integration tests (blocked by: T-A003)
    │
    ├── For each task, IB injects context:
    │   ├── Relevant business rules: "JWT must expire in 30d", "HMAC-SHA256 minimum"
    │   ├── Relevant requirements: T-132 criteria for authentication
    │   ├── Acceptance tests: pre-generated from T-133
    │   └── ADRs: "why we chose JWT over sessions"
    │
    ├── Agent executes with full context
    │
    └── IB runs acceptance tests
        ├── All pass → task.completed event emitted, next task unlocks
        └── Any fail → task.failed event, agent retries with failure context
```

## AgentTask Model

```go
// internal/domain/agent_task.go

type AgentTask struct {
    ID              uuid.UUID
    OrgID           uuid.UUID
    GoalID          uuid.UUID     // parent goal this task belongs to
    Title           string
    Description     string        // detailed task description
    Instructions    string        // step-by-step instructions for the agent

    // Context injected from knowledge base
    ApplicableRules []uuid.UUID   // business rules the agent must follow
    Requirements    []uuid.UUID   // requirements this task satisfies
    AcceptanceTests []uuid.UUID   // tests that must pass for completion

    // Execution
    Status          AgentTaskStatus // pending, running, verifying, completed, failed
    AssignedAgent   string          // agent identifier (claude-sonnet-4-6, etc.)
    Attempts        int
    LastAttemptAt   *time.Time
    CompletedAt     *time.Time

    // DAG
    BlockedBy       []uuid.UUID   // tasks that must complete first
    Blocks          []uuid.UUID   // tasks that become available after this one

    // Output
    Artifacts       []Artifact    // files created/modified by the agent
    TestResults     []TestResult  // results from acceptance test run
}

type AgentTaskStatus string
const (
    AgentTaskPending    AgentTaskStatus = "pending"
    AgentTaskRunning    AgentTaskStatus = "running"
    AgentTaskVerifying  AgentTaskStatus = "verifying"
    AgentTaskCompleted  AgentTaskStatus = "completed"
    AgentTaskFailed     AgentTaskStatus = "failed"
    AgentTaskBlocked    AgentTaskStatus = "blocked"  // awaiting dependencies
)
```

## Goal Decomposition

```go
// internal/agent/decomposer.go

type GoalDecomposer struct {
    ai      Provider
    rules   RuleStore
    reqs    RequirementStore
}

// Decompose takes a high-level goal and returns a DAG of AgentTasks.
// It queries the knowledge base for relevant context before generating tasks.
func (d *GoalDecomposer) Decompose(ctx context.Context, goal string, orgID uuid.UUID) ([]AgentTask, error) {
    // 1. Find relevant rules, requirements, and ADRs via semantic search
    context, err := d.buildContext(ctx, orgID, goal)

    // 2. Ask AI to decompose into tasks with dependencies
    prompt := buildDecompositionPrompt(goal, context)
    tasks, err := d.ai.Complete(ctx, prompt)

    // 3. For each task, attach applicable rules and requirements
    for i := range tasks {
        tasks[i].ApplicableRules = d.matchRules(ctx, tasks[i], context.Rules)
        tasks[i].Requirements   = d.matchRequirements(ctx, tasks[i], context.Requirements)
    }

    return tasks, nil
}
```

## Context Building

The quality of agent output is proportional to the quality of context injected.
IB builds context by:

1. Semantic search over rules: "what rules apply to this task description?"
2. Semantic search over requirements: "what requirements does this task address?"
3. Full ADR retrieval for the task's domain
4. Recent similar tasks: "how did we implement similar things before?"

```go
type AgentContext struct {
    Rules        []BusinessRule    // must + should rules for this task
    Requirements []Requirement     // requirements this task satisfies
    ADRs         []ArchDecision    // relevant architecture decisions
    PriorWork    []AgentTask       // similar completed tasks (few-shot examples)
}
```

## Verification

After agent completes:
1. Run acceptance tests from T-133 against the agent's artifacts
2. If all pass → emit `agent_task.completed`
3. If any fail → emit `agent_task.failed` with failure details → retry with failure as context
4. Max 3 attempts before human escalation

## Human-in-the-Loop

IB asks for human approval:
- Before dispatching the first task in a goal (show decomposition plan)
- When a task has been attempted 3 times without passing
- When a task's rules conflict with each other (ambiguous requirements)
- When the decomposer is uncertain (confidence < 0.8)

IB never runs autonomously without a human approving the initial plan.

## API

```
POST /api/v1/goals                    — create a high-level goal (triggers decomposition)
GET  /api/v1/goals/:id/tasks          — see decomposed tasks + status
POST /api/v1/goals/:id/approve        — human approves decomposition, dispatch begins
GET  /api/v1/agent-tasks/:id          — task detail + context + test results
POST /api/v1/agent-tasks/:id/retry    — manually retry a failed task
```

## Acceptance Criteria

- [ ] `AgentTask` entity with all fields
- [ ] `GoalDecomposer` creates a DAG of tasks from a high-level goal
- [ ] Context built from rules + requirements + ADRs via semantic search
- [ ] Human approval step before dispatch
- [ ] Agent receives context as structured system prompt
- [ ] Acceptance tests run after agent completion
- [ ] Pass → completed; fail → retry with failure context
- [ ] Max 3 attempts before human escalation
- [ ] All events emitted to T-120 event store
- [ ] 90% test coverage

## Dependencies

- T-120 (event sourcing — all state via events)
- T-131 (business rules — context injection)
- T-132 (requirements — task-to-requirement linkage)
- T-133 (test generation — acceptance tests for verification)
- T-023 (semantic search — context building)
- T-020 (AI provider — decomposition + agent execution)
