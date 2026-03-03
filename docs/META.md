# The Meta-Loop — IB Builds IB

> This document describes how Infinite Brain is used to build Infinite Brain.
> It is both a development methodology and the product's ultimate proof of concept.

---

## The Idea

The strongest possible demonstration of what Infinite Brain does is to use it to build itself.

Every feature spec in `features/*/README.md` is a knowledge node.
Every task in `docs/TASKS.md` is a task node.
Every architectural decision in `docs/ARCHITECTURE.md` is an ADR node.
Every business rule in `docs/CLAUDE.md` is a business rule node.
Every idea in `docs/BRAINSTORM.md` is a captured thought waiting for triage.

When these live in IB, the system can:
- Answer "what is the current status of authentication?" from the knowledge graph
- Detect conflicts between feature specs ("T-104 says X but T-120 says Y")
- Generate acceptance tests from the criteria in each spec (T-133)
- Identify gaps: "T-028 references T-023 but T-023 has no implementation tasks"
- Prioritize the backlog by True North alignment (T-151)
- Dispatch agents to implement tasks with full spec context (T-134)
- Verify completion when acceptance tests pass

**The product builds itself using its own features.**

---

## What Lives in IB (The Project Knowledge Graph)

| Source | Node Type | IB Feature Used |
|---|---|---|
| `features/*/README.md` | `requirement` | T-132 — Acceptance criteria linked to each spec |
| `docs/TASKS.md` | `task` | T-030 — Task management |
| `docs/ARCHITECTURE.md` | `decision` (ADR) | T-028 — Knowledge graph node type |
| `docs/CLAUDE.md` (rules) | `business_rule` | T-131 — Business rules with conflict detection |
| `docs/BRAINSTORM.md` | `note` | T-010 — Captured thoughts, triaged over time |
| `docs/IDEA.md` | `decision` (vision) | T-028 — Strategic decisions |
| This conversation | `note` → `decision` | T-010 + T-139 — Decision journal |
| ADR decisions | `decision` | T-028 — Architecture decisions |

---

## The Build Loop + Self-Improvement Loop

Once IB has a self-hosted instance and this project's knowledge is loaded:

```
BUILD LOOP
──────────
1. Select next task (by True North alignment × dependency order × coherence need)
        ↓
2. IB queries: relevant specs, business rules, architectural constraints
   + lessons from past retrospectives for this task category
        ↓
3. IB generates acceptance tests from the spec's criteria (T-133)
        ↓
4. IB creates an AgentTask with: spec + rules + tests + dependencies + past lessons (T-134)
        ↓
5. Human reviews decomposition and approves dispatch
        ↓
6. Agent executes implementation with full context
        ↓
7. Acceptance tests run — pass = continue; fail = agent retries with failure context
   (max 3 attempts before human escalation)
        ↓
8. IB marks requirement as implemented; updates task status; unlocks dependents
        ↓
9. IB detects new gaps created by this implementation (T-136)

SELF-IMPROVEMENT LOOP (wraps each build iteration)
───────────────────────────────────────────────────
10. Retrospective captured (T-168):
    - Retry count and failure patterns
    - Plan vs. actual task delta
    - True North alignment of output
    - Human corrections made
    - Business rules violated (present or missing)
        ↓
11. Pattern detection across retrospectives (T-169):
    - Has this failure pattern appeared before?
    - Is this a systemic gap or a one-off?
        ↓
12. If pattern threshold reached → methodology update candidate generated (T-170):
    - Decomposition prompt refinement
    - Test template update
    - Business rule addition (T-131)
    - Context retrieval scoring adjustment
        ↓
13. Alignment guard (T-171):
    - Does this update maintain True North alignment?
    - If yes → apply; version the old artifact; measure outcomes next iteration
    - If no → reject; flag for human review
        ↓
14. Next iteration inherits the improvement
```

**The invariant**: the system gets better with every iteration, and "better" means both
more efficient (fewer retries, more accurate decomposition) AND more aligned (True North
alignment of output). Efficiency without direction is faster entropy.

This loop is the product. Building IB with IB makes every iteration a live test of the
self-improvement mechanism itself.

---

## Current State (2026-04-01)

The knowledge graph has not yet been bootstrapped — we are building the engine.
All specs exist as markdown files in `features/` and `docs/`. These will be imported
when the first self-hosted IB instance runs (T-163).

Once T-163 is complete:
- All 162+ tasks become IB task nodes
- All feature specs become IB requirement nodes with acceptance criteria
- All business rules from CLAUDE.md become IB business rule nodes
- BRAINSTORM.md entries get triaged into the knowledge graph
- This document (META.md) becomes a strategic decision node

---

## The Entropy Problem for IB's Own Development

IB's codebase is itself a knowledge system subject to entropy:
- Specs get stale when implementation diverges from the original design
- Decisions made early get forgotten and re-made inconsistently
- The gap between "what we planned" and "what we built" widens silently

**IB fights its own entropy by feeding itself:**
- Every implementation lesson → captured node → feeds future agent context
- Every spec change → IB's requirement nodes update → tests regenerate
- Every architectural drift → IB detects it (T-137) → surfaces for correction

The coherence score (T-164) for the IB project itself should trend upward as the
knowledge graph is enriched. If it trends downward, entropy is winning — and we need
to inject more energy (documentation, decision capture, spec updates).

## The True North for IB's Own Development

IB's True North (T-150) for building itself:

**Honest objective**: Ship a production-quality, open-source intelligence platform that
demonstrates every tier of the recursive brain model, builds itself using its own features,
and serves as a portfolio artifact that shows architectural depth, security maturity,
and product clarity.

**Constraints**:
- Never ship incomplete features — acceptance criteria must pass before done
- Never compromise on security or compliance (SOC2 / HIPAA / EU AI Act)
- Every architectural decision must have a written ADR
- The meta-loop must be demonstrable: IB must visibly help build IB

**What success looks like**:
- A developer reads the codebase and says "this is how I want to build software"
- A company asks to use IB for their team and can self-host it today
- An investor sees a platform, not a note-taking app

---

## Feeding IB (Ongoing)

As we build, we feed IB:
- Every new decision made in a session → captured as a `decision` node
- Every new feature idea → captured as a `note`, triaged to `planned` or `someday`
- Every implementation lesson → captured as a `note`, tagged with relevant feature
- Every time a spec changes → IB's acceptance criteria update; tests regenerate

The goal: by the time T-134 (agent dispatch) is implemented, IB's knowledge graph is
rich enough that the agents can operate with genuine autonomy.

The product earns its autonomy by being well-fed.
