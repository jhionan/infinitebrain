# Infinite Brain — Brainstorm & Ideas

> This is a living document. Drop any idea here — no filtering, no judgment. We'll triage later.
> Both Rian and Claude contribute. Mark your ideas with **[R]** (Rian) or **[AI]** (Claude/research).

---

## How to Use This Document

- Add ideas freely in any section
- If an idea is strong enough to become a task, move it to `docs/TASKS.md` and link it here
- Tag ideas with `[maybe]`, `[hot]`, `[someday]`, or `[needs research]`
- Leave comments and questions inline — this is a dialogue

---

## THE SELF-IMPROVEMENT LOOP (2026-04-01) [R][hot]

> "We need a self-improve system. While in the loop of self-build we need to evolve in the
> right direction. We need to learn from error and improve methodology / efficiency in every
> iteration."
> — Rian, 2026-04-01

### The Build Loop vs. The Learning Loop [R]

The build loop (T-134, META.md) produces outputs.
The learning loop wraps the build loop and makes each iteration better than the last.

```
╔═══════════════════════════════════════════════════════╗
║  LEARNING LOOP                                        ║
║                                                       ║
║   ┌─────────────────────────────────────────────┐    ║
║   │  BUILD LOOP                                 │    ║
║   │                                             │    ║
║   │  Task → Context → Tests → Agent → Verify   │    ║
║   │                                ↓            │    ║
║   │              Pass / Fail + artifacts        │    ║
║   └──────────────────────┬──────────────────────┘    ║
║                          ↓                           ║
║              Retrospective capture                   ║
║              (what happened, what didn't work)       ║
║                          ↓                           ║
║              Pattern detection across iterations     ║
║              (this failure type has appeared 3x)     ║
║                          ↓                           ║
║              Methodology update                      ║
║              (prompt revised / rule added /          ║
║               decomposition granularity changed)     ║
║                          ↓                           ║
║              Next iteration inherits improvement     ║
╚═══════════════════════════════════════════════════════╝
```

### What the System Learns From [hot]

**Signal 1: Agent retry count**
Every extra attempt is a failure signal. What was missing?
- Context gap: a relevant rule wasn't injected
- Prompt ambiguity: the task description allowed a wrong interpretation
- Spec gap: acceptance criteria didn't cover an edge case
- Dependency miss: agent needed something that wasn't yet built

Action: capture the failure message → identify the gap category → fix upstream
(add missing rule, sharpen the criterion, update the decomposition template)

**Signal 2: Plan vs. actual delta**
If the decomposition said 4 tasks but implementation needed 7, the spec was too coarse.
If the decomposition said 6 tasks but only 3 were real, the spec was over-specified.
The delta between planned and actual is a precision metric for the decomposer.

Action: feed delta back into the decomposer's few-shot examples — next decomposition uses
successful past decompositions as reference

**Signal 3: Test quality — did tests catch real bugs?**
Tests that always pass are useless. Tests that block valid implementations are wasteful.
The right signal: bugs found post-merge that the tests didn't catch.

Action: those bugs become new acceptance criteria → test template updated → future
generated tests cover that class of failure by default

**Signal 4: True North alignment of output**
The system can build efficiently in the wrong direction. Speed without alignment is drift.
Every completed task is scored against True North after completion (T-151).

Action: if a whole iteration scores low on alignment, the decomposer's next prompt includes
a stronger True North filter. The system learns that fast + wrong is worse than slow + right.

**Signal 5: Business rule violations**
If the agent violated a rule that exists in IB but wasn't injected, the context retrieval
missed something. If the agent violated a rule that doesn't exist yet, capture it now.

Action: violated-but-missing rule → add to business rule base (T-131)
         violated-but-not-injected → update context retrieval scoring for that rule category

**Signal 6: Human corrections**
Every time a human overrides an AI decision, reviews an agent output and changes it,
or marks a completed task as "needs redo" — that is the highest-quality learning signal.
Human corrections are ground truth.

Action: corrections feed T-129 (personal AI preferences) at the individual level AND
the methodology evolution engine at the system level

### The Methodology Evolution Engine [hot]

Patterns in retrospectives → updates to durable methodology artifacts.

```go
// internal/learning/methodology.go

type MethodologyArtifact string
const (
    ArtifactDecompositionPrompt  MethodologyArtifact = "decomposition_prompt"
    ArtifactTestGenTemplate      MethodologyArtifact = "test_gen_template"
    ArtifactContextRetrievalRule MethodologyArtifact = "context_retrieval"
    ArtifactAcceptanceCriteria   MethodologyArtifact = "acceptance_criteria_format"
    ArtifactBusinessRule         MethodologyArtifact = "business_rule"
)

type MethodologyUpdate struct {
    ID              uuid.UUID
    Artifact        MethodologyArtifact
    PreviousVersion string
    NewVersion      string
    Rationale       string           // why this change was made
    TriggeredBy     []uuid.UUID      // retrospective IDs that drove this
    FailurePattern  string           // the recurring failure this fixes
    AppliedAt       time.Time
}
```

Every methodology update is:
- Versioned (we can roll back if an update makes things worse)
- Traced to the retrospectives that triggered it
- Measured: did outcomes improve after this update?

### The Right Direction Constraint [hot]

Efficiency without direction is just faster entropy. The learning loop must evolve toward
True North, not just toward faster execution.

**Dual optimization target:**
1. Efficiency: fewer agent retries, more first-attempt passes, better plan accuracy
2. Effectiveness: output alignment with True North, coherence delta per iteration

These can conflict. A fast path might be misaligned. A perfectly aligned path might be slow.
The system must learn to optimize both — and when forced to choose, alignment wins.

Implementation: every retrospective captures both metrics. The methodology evolution engine
only accepts updates that don't reduce True North alignment even if they improve efficiency.

### Learning Has Its Own Entropy [hot]

Lessons learned in iteration 5 can become obsolete by iteration 50.
A decision made early might be reversed. A business rule might change.
Old learning applied in the wrong context is worse than no learning.

The learning knowledge base is subject to the same freshness decay as all other nodes (T-167).
Outdated methodology artifacts decay and are surfaced for review.
If a retrospective lesson contradicts current business rules, IB flags the conflict.

The learning system maintains its own coherence.

### New Tasks

- **T-168**: Iteration retrospective capture — structured after each build cycle
- **T-169**: Pattern detection across retrospectives — recurring failures → methodology update candidates
- **T-170**: Methodology evolution engine — versioned updates to prompts/templates/rules; measured outcomes
- **T-171**: Alignment guard on learning — reject methodology updates that reduce True North alignment

---

## THE ENTROPY MODEL — IB AS ENERGY INJECTION (2026-04-01) [R][hot]

> "I don't want the hierarchy to be fixed. Mimic how nature works. Protons and electrons are
> unbalanced alone — they group into a bigger structure. Atoms → elements → molecules → cells
> → organs → tissue → organisms → families → communities → countries → trade blocs → planet
> → solar system → ... All elements seek balance, but chaos is the default. To keep balance
> we need to inject energy. I want Infinite Brain to be the energy we inject to keep everything
> balanced."
> — Rian, 2026-04-01

### The Core Physics [R][hot]

This is not a metaphor — it is the governing principle of the architecture.

**Entropy is the natural state of knowledge systems.**
- A thought not captured: lost
- A decision not recorded: re-made at cost
- A team's knowledge not organized: rediscovered forever
- An org's direction not maintained: drift and fragmentation

**Balance requires energy.** Order does not emerge spontaneously.
It must be actively maintained against the natural tendency toward disorder.

**Infinite Brain is the energy.**
At every level of any hierarchy, IB:
- Measures the coherence of the knowledge system (how close to balance?)
- Surfaces where entropy is winning (gaps, disconnected nodes, stale knowledge, concentration risks)
- Injects intelligence to restore balance (connections, insights, tasks, True North recalibration)

### The Open Hierarchy [R][hot]

**Fixed level names are wrong.** Nature doesn't have a fixed number of organizational levels.
The hierarchy emerges from the structure of the problem.

The `org_units` table is a free-form tree:
- `parent_unit_id` is the only structural constraint
- `name` is user-defined: "cell", "pod", "tribe", "chapter", "galaxy" — whatever fits
- No `unit_type` enum — or if present, it's just a label with no constraint on values
- `depth` is computed from the tree, not stored

This means:
- A solo developer has `[root] → [me]`
- A startup has `[company] → [team] → [person]`
- A DAO has `[network] → [guild] → [circle] → [member]`
- A city has `[city] → [district] → [neighborhood] → [block] → [household] → [person]`

All use the same engine. The brain applies at every node.

### Coherence Score — The Core Metric [hot]

If IB's job is to maintain balance, it needs to measure balance.

**Coherence score** = how ordered/balanced is this unit's knowledge system?

```go
type CoherenceScore struct {
    UnitID              uuid.UUID
    ComputedAt          time.Time
    Score               float64   // 0.0 (chaotic) to 1.0 (coherent)

    // Components
    KnowledgeDensity    float64  // captured knowledge vs. estimated gaps
    ConnectionDensity   float64  // ratio of connected to isolated nodes
    TrueNorthAlignment  float64  // % of work aligned with declared True North
    DecisionCoverage    float64  // decisions with documented rationale / total decisions
    KnowledgeFreshness  float64  // recency of knowledge (old unreviewed nodes = entropy)
    ConcentrationRisk   float64  // inverse of single-person knowledge dependencies
    ContributionBalance float64  // knowledge spread across unit members (no silos)
}
```

**When coherence is low, IB acts:**
- `KnowledgeDensity` low → "These 7 areas have no captured knowledge — they're entropy risks"
- `ConnectionDensity` low → "42 isolated nodes could be linked; surfacing candidates"
- `TrueNorthAlignment` low → "60% of current work is misaligned with your True North"
- `ConcentrationRisk` high → "If João left today, 14 processes would have no documented owner"
- `KnowledgeFreshness` low → "38 nodes haven't been reviewed in 6+ months — context may be stale"

This is IB injecting energy to restore balance. Not a dashboard — an active intervention.

### The Energy Metaphor Applied to Features [hot]

Every major IB feature is energy injection at a specific entropy failure mode:

| Entropy failure | IB energy injection |
|---|---|
| Thoughts not captured | Zero-friction capture (voice, bot, email, webhook) |
| Captured knowledge not organized | AI classification + auto-tagging + PARA |
| Knowledge not connected | Insight linker + knowledge graph edges |
| Decisions not recorded | Decision journal + ADR nodes |
| Context lost between sessions | "Where was I?" + breadcrumb trail |
| Drift from True North | Alignment scoring + drift detection |
| Knowledge concentrated in one person | Concentration risk detection + knowledge transfer |
| Ideas suppressed by hierarchy | Idea pipeline (anonymous, merit-based) |
| Work misaligned with strategy | True North prioritization |
| Requirements not verified | Test generation + agent verification |

Each feature fights entropy at a specific point in the knowledge lifecycle.

### What This Means for the Product [hot]

**IB is not a feature set — it's a coherence engine.**

The question for any new feature is not "what does it do?" but:
"At what scale does it fight entropy? What kind of imbalance does it restore?"

Features that don't answer that question don't belong in the product.

**The coherence score is the north star metric for the product:**
- Not DAU, not notes created, not AI calls
- Coherence delta: did the user's/team's/org's knowledge system become more coherent this month?

**The product grows with the hierarchy's needs:**
- When a unit becomes incoherent, IB notices and acts
- When a unit grows and needs to split, the tree extends
- When two units merge, their knowledge graphs merge and IB finds the cross-connections

**New Tasks from This Model**

- **T-164**: `CoherenceScore` entity — compute and store per unit on a schedule
- **T-165**: Coherence dashboard — per-unit coherence breakdown with "what to fix" actions
- **T-166**: Open unit hierarchy — remove `unit_type` enum constraint; user-defined names at any depth
- **T-167**: Entropy alerts — notify when coherence drops below threshold (e.g., "your team's knowledge freshness is falling")

---

## EU AI ACT COMPLIANCE ANALYSIS (2026-04-01) [AI][hot]

> EU AI Act (Regulation 2024/1689) — effective Aug 2024.
> World's first comprehensive risk-based AI law. Extraterritorial: applies if output used in EU.
> AGPL-3.0 open-source self-hosted IB is largely exempt from high-risk obligations.
> infinitebrain.io hosted SaaS is NOT exempt.

### IB Features Mapped to Risk Tiers

**PROHIBITED — must never build, must technically block**

| What's prohibited | IB risk | Mitigation |
|---|---|---|
| AI social scoring (governments) | Low — we're not government | Not applicable |
| AI that manipulates behavior subliminally | Medium — ADHD nudges could drift here | Nudges must be transparent, consensual, opt-out |
| AI exploitation of vulnerabilities (disability, age) | Medium — ADHD users are a protected group | Nudges help users achieve their own goals, not external goals; never coercive |
| Real-time biometric ID in public spaces | Not applicable | N/A |

**ADHD manipulation boundary**: IB's nudges (hyperfocus guard, focus timer, distraction capture)
assist users in pursuing their own stated goals. This is categorically different from manipulation.
The legal test: who benefits? If the user benefits → assistance. If a third party benefits at the
user's expense → manipulation. IB must always be on the right side of this line.

**HIGH-RISK — requires strict controls if IB is used in these contexts**

The high-risk classification applies to the *use case*, not just the tool. IB features that
touch the following domains create high-risk obligations if deployed in those contexts:

| Domain | IB Feature | Risk if used for... |
|---|---|---|
| Employment (Annex III §4) | Org expertise graph (T-140), org health (T-144), contribution risk (T-143) | Hiring, firing, performance evaluation, task assignment to employees |
| Education (Annex III §3) | AI coaching, skill gap detection | Academic assessment, grading, admission decisions |
| Healthcare (Annex III §5) | AI recommendations on PHI nodes | Clinical decisions, diagnosis support |

**The org tier is the critical risk zone.**
If an employer uses IB's org brain to evaluate employee performance → IB is a high-risk AI system.
The technical controls we already designed (T-146 anonymization, T-149 no individual queries)
are the legal defense. They must be technically enforced, not just policy.

**LIMITED RISK — transparency requirements (must do now)**

These apply to all IB AI features regardless of tier:

| Requirement | IB Feature | What must change |
|---|---|---|
| Bots must identify as AI | Telegram (T-070), WhatsApp (T-071), Slack (T-072) | Opening message must state: "I'm Infinite Brain, an AI assistant" |
| AI-generated content must be labeled | All AI outputs: classify, tags, insights, summaries, agent tasks | Every AI-generated node/response must carry `ai_generated: true` + model name |
| Users must be able to override AI decisions | All AI classifications (T-021, T-022) | Already in design via T-129 (corrections); must be surfaced clearly in UX |
| Right to explanation | AI prioritization, classification, scoring | "Why did IB recommend this?" must return a human-readable explanation |

**MINIMAL RISK — no additional obligations**

| Feature | Risk tier |
|---|---|
| Spam/noise filtering in inbox | Minimal |
| PARA classification (T-021) | Minimal (individual use) |
| Auto-tagging (T-022) | Minimal |
| Semantic search (T-023) | Minimal |
| Semantic dedup (T-128) | Minimal |
| True North alignment scoring (T-151) — personal | Minimal (user's own goals) |

### GPAI Deployer Obligations (Aug 2, 2025 deadline)

IB uses Claude (Anthropic's GPAI). Anthropic bears the GPAI *provider* obligations.
IB is a GPAI *deployer* — lighter obligations, but still real:

1. **Must not use GPAI in prohibited ways** — IB must not instruct Claude to do anything
   the EU AI Act prohibits (social scoring, manipulation, etc.)
2. **Must document AI usage** — which models, for what tasks, with what safeguards
3. **Must maintain usage policies** consistent with Anthropic's model card and ToS
4. **Must provide AI transparency** to end users (already required under Limited Risk)

This means IB needs an **AI usage register**: a machine-readable record of every AI operation,
which model was used, what data was sent (category, not content), what safeguards were applied.
This register is evidence in an audit.

### Open-Source Exemption Analysis

AGPL-3.0 IB is largely exempt from high-risk obligations (Article 2(12)) — **but only if:**
- The system is not itself high-risk
- The open-source provider doesn't receive revenue for the system (→ cloud hosting changes this)

| Deployment | Exempt? | Notes |
|---|---|---|
| Self-hosted IB, personal use | Yes | Article 2(12) exemption applies |
| Self-hosted IB, commercial org use | Partial | If used for employment decisions → deployer bears high-risk obligations |
| infinitebrain.io hosted SaaS | No | Commercial provider; full Act applies |

**Strategic implication**: The AGPL license is not just a community moat — it's a legal shield
for individual users. infinitebrain.io must bear the full compliance burden; self-hosters get
the exemption. This makes the hosted tier *more expensive* to operate (compliance overhead)
but *more valuable* to enterprise buyers (they don't have to manage compliance themselves).

### What Must Be Built

- **T-154**: EU AI Act compliance documentation — AI usage register, model card, risk assessment
- **T-155**: AI transparency labeling — `ai_generated`, `ai_model`, `ai_confidence` on all AI outputs
- **T-156**: Right to explanation — "why did IB do this?" for any AI decision
- **T-157**: Employment use prohibition — technical blocks + ToS on using org features for HR decisions
- **T-158**: Bot AI identity disclosure — all bots introduce themselves as AI on first message

### What We Must NEVER Build (EU AI Act Prohibited)

- Individual employee productivity scores visible to management
- Automated employment recommendations based on IB data
- Behavioral profiling of employees without explicit consent
- Any feature that ranks or scores individuals for third-party use
- Real-time monitoring of employee activity
- Inferred emotional states or psychological profiles of employees

**These are now both ethical commitments AND legal requirements.**

---

## TRUE NORTH: THE ALIGNMENT ANCHOR (2026-04-01) [R][hot]

> "If companies and individuals give a true north — company vision, objectives etc (and they
> need to mean what they want: profit, sustainability, etc) — it makes it easy to put everything
> on the same page. But with enough data it can be able to manage tasks for people and for AI."
> — Rian, 2026-04-01

### The Core Insight [R]

Every productivity system fails for the same reason: it tracks activity, not alignment.
You can be very busy and moving in the wrong direction. Infinite Brain solves this at the root.

**True North** is the honest declaration of what a person or organization is actually optimizing for.
Not the polished mission statement — the real one.

A company's True North might be:
- "Become profitable within 18 months (current runway is 14 months)"
- "We want to build something sustainable that we're proud of, not a VC exit"
- "We want to 10x revenue without hiring more than 12 people"

A person's True North might be:
- "I want to be a senior engineer at a company I respect within 2 years"
- "I want to work fewer than 45 hours a week and still ship great work"
- "I want to understand distributed systems deeply — not just use the tools"

These are fundamentally different optimization targets. A system that doesn't know your True
North can't tell you whether you're wasting your time or not.

### True North as the Root Node [hot]

In the knowledge graph, True North is a special node type: the root of everything.

```
True North (the honest objective)
    ├── Values (what you won't compromise)
    ├── Goals (time-bound milestones toward True North)
    │   ├── Projects (work that advances a goal)
    │   │   └── Tasks (what to do today)
    │   └── Areas (ongoing responsibilities)
    └── Anti-patterns (what to stop doing)
```

Every node in the graph has an implicit edge to True North — either it advances it, is neutral,
or works against it. IB can score this alignment and surface the conflicts.

### Alignment Scoring [hot]

With enough data, IB can evaluate any proposed task or project:

```
Proposed: "Build a detailed analytics dashboard for the admin panel"

IB's analysis:
  True North: "become profitable in 14 months"
  Current revenue bottleneck: "conversion rate, not retention"
  Alignment score: 0.3 (low — this doesn't affect conversion)

  Suggestion: "The dashboard doesn't move your revenue needle.
  Your True North says profitability. The 3 highest-aligned tasks right now are:
  - Fix the checkout friction (conversion impact: high)
  - Reduce churn in month 2 (retention: medium)
  - Ship the feature sales keeps losing deals over (revenue: high)"
```

This is the "what actually moves the needle" feature from the org intelligence tier —
but applied at the personal and team level too.

### Honest Objectives vs. Stated Objectives [hot]

IB captures both layers:
- **Stated objective**: "We want to build the best product in the market"
- **Real constraint**: "We need to hit $1M ARR before the runway runs out in June"

The real constraint governs decisions. IB holds both and surfaces when a decision is
optimizing for the stated objective but working against the real constraint.

This is uncomfortable to confront — which is exactly why most tools don't do it.
IB doesn't judge. It just makes the trade-off visible.

### Task Management Driven by Alignment [hot]

Once True North is established and there's enough captured data:

**For humans:**
- Daily task prioritization considers True North alignment, not just urgency/importance
- "This task has been postponed 5 times — is it still aligned with where you want to go?"
- Weekly review: "You spent 60% of your time on tasks with <0.4 alignment score"
- IB proposes tasks the person hasn't thought of that would advance True North

**For AI agents (T-134):**
- Every agent task is evaluated against True North before dispatch
- Agents that complete work misaligned with True North trigger a review
- The agent decomposition prompt includes: "Given this True North, here is what matters most"
- Goal decomposition always starts with: "Does this goal advance the True North?"

### Anti-patterns and Drift Detection [hot]

IB learns what consistently wastes time or moves away from True North:
- "You keep starting documentation projects but they rarely ship"
- "Meetings on Monday mornings consistently result in low task completion that week"
- "Time spent on feature X has never correlated with customer retention"

When a new task or pattern resembles a known anti-pattern, IB flags it.
Not to block — but to make the choice conscious.

### For Organizations: Mission Integrity [hot]

Companies frequently drift from their stated mission as they scale.
IB can detect this at the knowledge level:

- The mission says "customer-first" — but the last 20 decisions prioritized engineering velocity
- The values say "sustainable pace" — but 8 engineers have flagged overload this quarter
- The OKR says "reduce churn" — but 90% of shipped features are new acquisition features

IB surfaces the gap between the declared True North and the actual pattern of decisions.
Leadership sees the drift before it becomes a cultural crisis.

### New Tasks from This Design

- **T-150**: `TrueNorth` node type — honest objectives, values, constraints, anti-patterns
- **T-151**: Alignment scoring engine — score any task/project against True North
- **T-152**: Drift detection — surface gap between declared True North and actual decision patterns
- **T-153**: Alignment-driven task prioritization — prioritize by True North alignment, not just urgency

---

## PRIVACY ARCHITECTURE + SAFE CONTRIBUTION DESIGN (2026-04-01) [R][hot]

> "For personal use, all knowledge flows freely. For business, we need explicit permissions.
> Personal info inside the company layer must be PII-protected. We collect all data but we
> don't want to identify individuals, benchmark people, or discriminate. We need to create a
> safe, healthy environment where everyone contributes and nobody is punished. We need a layer
> to digest ideas, surface good ones, and discard bad ones."
> — Rian, 2026-04-01

### The Three Privacy Contracts [R][hot]

Each tier has a fundamentally different contract with the user.
**Mixing them up is a product killer.**

| Tier | Who sees data? | Attribution? | User controls? |
|---|---|---|---|
| Personal | Only you | Full — it's your brain | You own everything |
| Team | Your team | Full — small trusted group | You explicitly publish |
| Org | Nobody directly — only patterns | Stripped before any output | You contribute; aggregation protects you |

The personal → team transition is **explicit publish** (user action, not default).
The personal → org contribution is **de-attributed at ingestion** (can't be reversed).

### The Technical Privacy Model for Org Layer [hot]

**k-anonymity enforcement**
Every org-level query must return results from at least k=5 individuals.
If fewer than 5 people contributed to an insight, it is suppressed.
This prevents re-identification: even if you know 3 people in the org, you can't tell which one said X.

```go
// internal/org/anonymizer.go

type OrgInsightQuery struct {
    MinGroupSize int  // k-anonymity threshold, default 5
    Aggregated   bool // always true for org queries
}

func (s *OrgIntelligenceService) QueryInsights(ctx context.Context, q OrgInsightQuery) ([]OrgInsight, error) {
    if !q.Aggregated {
        return nil, apperrors.ErrForbidden // never return individual data
    }
    results := s.repo.QueryAggregated(ctx, q)
    // Strip any result where contributor count < MinGroupSize
    return filterBelowThreshold(results, q.MinGroupSize), nil
}
```

**Individual queries are rejected at the API layer**
No endpoint exists for "what did user X contribute to the org brain."
This is enforced at the service layer, not just by policy.

**Anonymization at contribution time**
When a user publishes to the org layer, name/ID is stripped immediately.
The contribution record stores only: `org_id`, `topic_embedding`, `category`, `timestamp_week` (not exact).
Exact timestamp is bucketed to week to prevent timing-based re-identification.

**Differential privacy for metrics**
Org-level numeric metrics (ideas submitted, topics raised) add calibrated noise before surfacing.
Prevents inference attacks: "the number changed by exactly 1 after João's meeting" → João said it.

### The Idea Pipeline Design [R][hot]

This is the most psychologically important feature in the entire org layer.
The system must make people WANT to share, not fear sharing.

**Anonymous submission (to org; visible to self)**
- User submits an idea in their personal brain
- "Share to org idea pool" is a voluntary action
- Once shared: contributor identity is stripped
- The idea now belongs to the collective, not the individual

**Deduplication and clustering**
- IB embeds the idea and checks for semantic similarity against existing ideas
- Similarity > 0.85: merged into existing cluster, +1 signal weight
- "6 people independently had this same idea" is far more powerful than one person saying it once
- This is the collective signal: ideas that emerge independently multiple times are high-signal

**Priority scoring**
- IB scores ideas against: current business gaps (T-136), stated OKRs, known customer pain
- An idea that addresses a known gap scores higher automatically
- This is not a popularity contest — a single well-timed idea can outscore 10 noise ideas

**Surfacing without attribution**
- Weekly digest to relevant stakeholders: "5 ideas this week worth exploring"
- Ideas presented without any human name attached
- Context: "this idea connects to the churn problem we've been tracking"

**Outcome tracking**
- If an idea is explored and leads to a change: the cluster is marked as "acted on"
- Future contributors can see: "this type of idea has led to action before"
- This creates a feedback loop: what kinds of ideas move the org?
- Individual contributors can see (only for themselves): "your ideas have been acted on 3 times"
  → Private positive feedback; never exposed to the org

### What We Must NEVER Build [R]

These features would destroy psychological safety and betray the product's principles:

- Individual contribution leaderboards
- "Most active contributor" badges
- Manager dashboard showing individual activity
- "Who has submitted the most ideas" views
- Any metric that lets a manager evaluate an employee based on IB data
- Automatic "flag this person's contribution as low quality" based on acceptance rate
- Any feature that makes an employee think: "if I share this, I might get fired"

**This is not just ethics — it is product strategy.**
The org brain is only as good as what people put into it.
If people are afraid, they contribute noise or nothing.
Fear kills the data flywheel.

### The Safe Space Design Principles [R]

1. **Contribution is always voluntary** — IB never auto-publishes personal knowledge to the org
2. **Identity is stripped at the org layer** — technically enforced, not just policy
3. **Ideas are evaluated on merit** — no votes, no likes, no names
4. **Nobody is penalized for bad ideas** — ideas are parked, not rejected with attribution
5. **The org gets smarter, individuals are not ranked** — collective intelligence only
6. **Employees can see their own contributions** — private personal feedback loop
7. **Consent is revocable** — you can delete your org contributions at any time (T-104 right to erasure)

### New Tasks from This Design

- **T-145**: Knowledge visibility model — `personal / team / org` scopes with explicit publish actions
- **T-146**: Org-layer anonymization — strip attribution at ingestion; k-anonymity enforcement
- **T-147**: Idea pipeline — anonymous submission → semantic clustering → priority scoring → surfacing
- **T-148**: Differential privacy for org metrics — add calibrated noise; prevent timing attacks
- **T-149**: Org insights query API — aggregated only; group size enforcement; no individual queries

---

## THREE-TIER PLATFORM VISION (2026-04-01) [R][hot]

> "Individual: dump your brain into the tool, it helps with decisions, remembers meetings and
> people. Company: all employees dump their work data, the brain organizes and improves the
> whole organization, identifies core metrics, understands what moves the needles, identifies
> gaps, and ultimately becomes a framework for organization — a project manager at company
> level, not just product level."
> — Rian, 2026-04-01

### Tier 1: Personal Memory Augmentation [R]

The core individual use case, broader than ADHD:
- "What did I discuss with João in the March meeting?" — IB knows, because you logged it
- "What were my reservations about switching to Postgres?" — IB has the decision log
- "Should I take this job offer?" — IB surfaces your past thoughts on priorities, constraints, tradeoffs
- "Who do I know at this company?" — IB has the relationship graph with interaction history

**The key insight**: IB is not a search engine over your notes. It is an active participant in
your decisions that knows everything you've captured and can surface the right context unprompted.
The difference: you don't need to remember what to search for — IB connects the dots for you.

**Relationship Graph** [hot]
- Every person you interact with becomes a Contact node
- Every meeting, email, call, message creates an Interaction edge
- IB extracts: topics discussed, commitments made, follow-ups promised
- "Prepare me for my meeting with Sarah next Tuesday" → IB returns: last 5 interactions,
  open commitments, topics she cares about, how her projects are going
- This is the world's best CRM, but you never have to fill it in manually

**Decision Journal** [hot]
- Every major decision captured with: context, options considered, rationale, outcome
- IB resurfaces decisions when relevant: "you made a similar choice 8 months ago — it went X"
- Pattern detection: "you consistently underestimate scope on frontend work"

### Tier 2: Team Brain [R]

When a team shares an IB workspace:
- **Expertise graph**: IB infers who knows what from their captured knowledge (not org charts)
  - "Who on the team has dealt with Stripe webhooks before?" → IB knows
- **Decision continuity**: team decisions persist even when people leave
  - "Why did we choose this architecture?" → IB has the full discussion, not just the outcome
- **Cross-person connection**: "What João said in standup connects to what Ana captured last week"
  - IB surfaces these without anyone having to schedule a meeting
- **Onboarding acceleration**: new hire asks IB questions instead of interrupting senior engineers
  - IB knows the codebase, decisions, patterns, and exceptions

### Tier 3: Organizational Intelligence [R][hot]

This is the moonshot. All employees contribute. IB synthesizes across the whole org.

**What moves the needle** [hot]
- IB ingests: deal data, support tickets, feature requests, sprint velocity, customer feedback, hiring
- It identifies correlations humans can't see across silos:
  - "Deals close 3x faster when sales engineers join in week 1 — but only 20% of deals have them"
  - "Every time the team ships >5 features in a week, support tickets spike 3 weeks later"
  - "The customers who churn in month 4 all had the same onboarding rep"
- These are real leading indicators, not dashboard vanity metrics
- IB doesn't just surface them — it explains the causal chain from the underlying data

**Organizational Gaps** [hot]
- IB detects what the org doesn't know it doesn't know:
  - Nobody owns customer success post-trial — it falls between sales and support
  - Two teams are building the same internal tool
  - A compliance requirement changed 3 months ago but only one team updated their process
  - The runbook for this critical process lives only in one person's head

**Knowledge Preservation** [hot]
- Every employee's captured knowledge feeds the org brain
- When someone leaves: their knowledge is already in IB, not in their head
- IB can identify "knowledge concentration risk": people who are single points of failure
  - "If Rian left today, these 12 processes would have no documented owner"
- Automatic knowledge transfer: IB generates a handoff document from the departing person's nodes

**Real Org Chart vs. Paper Org Chart** [needs research]
- IB maps who actually communicates with whom, who unblocks whom, who creates vs. consumes
- Often very different from the org chart
- This is genuinely valuable for identifying future leaders and organizational bottlenecks

**Organizational Health Metrics IB Could Define** [hot]
- Decision velocity: how fast do decisions get made and acted on?
- Knowledge flow: does information cross team boundaries, or silo?
- Meeting ROI: for every meeting captured, what decisions came out of it?
- Context loss rate: how often do teams re-discover known information?
- Dependency concentration: which people or systems are bottlenecks?

### New Tasks from Org Intelligence

- **T-138**: Relationship/interaction graph — people nodes + interaction edges; automatic extraction
- **T-139**: Decision journal — decisions with context, rationale, outcome; pattern detection
- **T-140**: Expertise graph — infer who knows what from captured knowledge
- **T-141**: Org-level knowledge synthesis — cross-user insights and connections
- **T-142**: Needle-mover analysis — correlate internal activity to business outcomes
- **T-143**: Knowledge concentration risk — single points of failure detection
- **T-144**: Organizational health dashboard — decision velocity, knowledge flow, meeting ROI

---

## THE BIG PIVOT: Intelligent Domain Database (2026-04-01) [R][hot]

> "This is a super database on steroids. Why can't it act as a project manager / product manager
> for projects — where we push documentation and business rules in, and it understands all the
> logic, orchestrates agents, creates small tasks with clear requirements and tests that must
> pass to prove the task is done? We could use our software to help us build our software."
> — Rian, 2026-04-01

### The Core Realization [R]

Infinite Brain is not just a PKM tool. It is a **domain intelligence engine**. When you push your
documentation and business rules into it, it doesn't just store them — it understands them,
can reason about them, and can use them to drive autonomous execution.

This changes the product at the core: from "capture everything, forget nothing" to
**"understand everything, execute correctly."**

### What This Enables

**Business Rules as First-Class Citizens** [hot]
- A `BusinessRule` node type: name, description, version, owner, conflicts_with[]
- Rules are queryable: "what are all rules that affect user authentication?"
- Rules are versioned via event sourcing (T-120) — when did this rule change and why?
- Rules can conflict: IB detects "you added a rule that contradicts this existing one"
- Every task carries the relevant rules as context — agents can't ignore them

**Requirement → AcceptanceCriteria → Test Triad** [hot]
- Every requirement stored in IB has acceptance criteria attached
- Acceptance criteria are machine-verifiable: IB generates the tests from the criteria
- When an agent completes a task, the pre-generated tests run
- Tests passing = task is proven done, not just "looks done"
- This is TDD at the product level, not just the code level

**AgentTask Orchestration** [hot]
- IB decomposes a high-level goal into concrete `AgentTask` nodes
- Each AgentTask has: description, business rules injected, acceptance tests, dependencies
- IB dispatches agents with full knowledge base context
- Agents report back; IB verifies; next task unlocks
- You stay in the loop for strategic decisions; IB executes the tactical

**The Meta-Product Loop** [hot]
- We use IB to build IB
- All `features/` specs, TASKS.md, ARCHITECTURE.md are nodes in IB
- When we add a new feature spec, IB can generate implementation tasks from it
- IB can detect gaps: "you have a requirement for rate limiting but no task for it"
- IB can generate tests from the acceptance criteria in the feature specs
- **Best possible portfolio demo**: the product proves itself by building itself

**Context as a Service** [hot]
- Other AI tools (Cursor, Claude Code, GitHub Copilot) can query IB for domain context
- API endpoint: `GET /api/v1/context?query=how+should+auth+work`
- Returns: relevant business rules, ADRs, requirements, past decisions
- Eliminates hallucination from AI coding tools — they have the ground truth
- Monetization angle: teams pay for this as an API subscription

### New Task Ideas from This Insight

- **T-131**: `BusinessRule` node type + conflict detection
- **T-132**: `Requirement` → `AcceptanceCriteria` structured linking
- **T-133**: Test generation from acceptance criteria (AI generates test stubs from criteria)
- **T-134**: `AgentTask` entity + agent dispatch loop
- **T-135**: Context API endpoint for external AI tools
- **T-136**: Gap analysis — IB detects uncovered requirements in the codebase
- **T-137**: Architecture drift detection — code diverges from documented intent

### Open Questions [needs research]
- Where is the boundary between IB as a PKM and IB as a dev platform?
  → Probably: personal IB is the personal knowledge layer; Project Brain is the team/dev layer
  → They share the same engine; different UX surfaces
- Does this compete with Linear, Jira, or complement them?
  → Complement: IB holds the WHY; Linear holds the ticket flow
  → IB can sync tasks to Linear, but the context lives in IB
- Can acceptance criteria actually be machine-verifiable for all task types?
  → For code tasks: yes (generate test stubs from criteria)
  → For design/content tasks: partial (AI judges against criteria)
  → For research tasks: no (human review required)

---

## 0. MVP Scope Decision (2026-03-04)

**Decision**: Scope cut to the core loop — capture, process, prioritize, focus.

Everything in this document is valid brainstorm material. Most of it is `[someday]`. The items marked `[hot]` are strong candidates for post-MVP v1.1.

**Architecture clarification (2026-03-04)**: MVP output is a REST API — no web or mobile app. Telegram, WhatsApp, and Slack bots ARE the user interface. Email capture and webhooks are also first-class capture interfaces. Users live in these tools already; meet them there.

**What moved to `someday`** (parked, not dead):
- Any web or mobile frontend
- Calendar sync and scheduling
- Contacts / relationship CRM
- Gamification engine (XP, streaks, boss battles)
- Body doubling / social features
- Apple Watch + haptics
- Hyperfocus guard (T-041), energy scheduling (T-043)
- Weekly review (T-026)
- GitHub, Readwise integrations (T-073–074)
- Observability, rate limiting

**MVP tasks** are tracked in `docs/TASKS.md` (Tiers 1–5 in the MVP plan).

---

## 1. Core ADHD Pain Points We Must Nail

> These are the non-negotiables. If we fail here, the app fails.

### Time Blindness [AI]
- ADHD brains have no natural sense of time passing — "time is now or not now"
- Features to combat this:
  - Visible countdown timers on active tasks (not just in a corner — prominent, ambient)
  - "You've been on this for 47 minutes" gentle nudges
  - Time estimates shown alongside tasks, with a running gap tracker ("you said 30 min, it's been 1h 12m")
  - Visual time bars that shrink in real-time while working

### Task Initiation (The Starting Problem) [AI]
- Many ADHD people know what to do but can't start — dopamine gap at initiation
- Ideas:
  - "Micro-first-step" AI: when you add a task, AI auto-generates the tiniest possible first action ("open the file", "write one sentence")
  - "Just 5 minutes" mode — commit to only 5 min, no pressure to continue
  - Task warm-up sequence: before big tasks, show a 3-step micro-ramp
  - Friction-removing shortcuts: one tap to "start" that opens the right app/file/document automatically

### Decision Fatigue [AI]
- ADHD users get paralyzed by too many choices
- Ideas:
  - Single "What do I do now?" button — AI picks the best next task based on energy, time available, and priorities
  - Daily "3 things" mode — AI selects your top 3 tasks for the day, hide the rest
  - Automatic inbox triage: AI suggests actions for everything in Inbox, you just approve/reject

### Hyperfocus & Rabbit Holes [AI + existing feature]
- Already planned: Hyperfocus Guard (T-041)
- Additional ideas:
  - "Are you still on track?" pop-up after 90 min on same task
  - Website/app usage integration (screen time API) to detect hyperfocus on distracting content
  - "Rabbit hole tracker" — log the detour so you can revisit intentionally later

### Context Loss Between Sessions [AI + existing feature]
- Already planned: "Where Was I?" (T-046)
- Additional ideas:
  - Auto-generated "resume brief" at session start: what you were doing, last decisions made, next step
  - Breadcrumb trail: AI logs the last 5 things you touched before you closed the app
  - "Morning briefing": daily AI summary of open loops, scheduled tasks, and unfinished work from yesterday

---

## 2. Capture Ideas (Getting Stuff Into the Brain)

### Zero-Friction Capture [AI]
- Voice is fastest — Whisper transcription is already planned (T-011)
- Ideas:
  - Single tap from Apple Watch face to start voice capture
  - "Distraction capture" widget: lock screen widget that logs a thought without unlocking phone
  - iMessage integration: text yourself → it auto-imports
  - Screenshot → AI extracts text, context, and creates a note automatically
  - Photo of whiteboard/sticky note → OCR + note creation

### Ambient Capture [AI]
- Capture that happens without deliberate action:
  - Browser extension that auto-saves all visited articles to "Readings" area
  - Auto-log of opened files/documents with optional notes
  - Meeting notes auto-created from calendar events (paste transcript or AI generates agenda+notes template)

### Capture Sources [existing + new ideas]
- Already planned: WhatsApp (T-071), Telegram (T-070), Email (T-013), Webhooks (T-014)
- New ideas:
  - **iMessage** capture (share extension)
  - **Twitter/X** bookmarks sync
  - **YouTube** save to watch later with AI summary
  - **Spotify podcast** timestamps (bookmark a moment in a podcast)
  - **Apple Notes** import / watch for new notes
  - **Bear / Obsidian** two-way sync
  - **Linear / Jira** task sync for developers
  - **Github projects** task sync for developers

---

## 3. AI Features (Beyond the Basics)

### Already Planned
- Auto-classify (PARA routing) — T-021
- Auto-tagging — T-022
- Semantic search — T-023
- Q&A over knowledge base — T-024
- Daily/weekly digest — T-025, T-026
- Context restoration — T-027

### New AI Ideas [AI]

#### Proactive Surfacing
- "You haven't touched this project in 12 days" — gentle nudge
- "You captured 5 things about X this week — should we create a project?" — pattern detection
- "This task has been postponed 3 times — is it still relevant?" — stale task cleanup
- Connections between notes: "This new note relates to what you wrote 3 weeks ago about..."


#### Mood & Energy Awareness
- Ask daily: "How's your energy today? (High / Medium / Low)" — 1 tap
- AI adjusts task recommendations based on energy state
- Detect stress signals in writing (lots of "can't", "stuck", "overwhelmed") and offer a break
- Weekly mood/energy chart — correlation with productivity

#### AI Coach Mode [hot]
- Brief daily check-in: "What's your #1 priority today?"
- End-of-day review: "Did you hit it? What got in the way?"
- Weekly pattern insights: "You do your best work on Tuesday mornings — protect that time"
- Not a chatbot — structured micro-interactions, very low friction

#### "Smart Chunking" [hot]
- Input: a project or big task
- Output: AI breaks it into time-boxed chunks with realistic time estimates
- Arranges chunks into a weekly schedule based on your available calendar slots
- Handles "this task depends on that one" ordering automatically

#### Emotional Support Layer [needs research]
- Detect when user is in a frustration/overwhelm loop
- Offer CBT-lite reframes: "You've been stuck for 30 min — what's one thing that's blocking you?"
- Celebrate wins explicitly (ADHD brains often miss their own wins)

---

## 4. Gamification & Dopamine [AI]

> Research shows 48% higher retention and 60% compliance boost with gamification for ADHD.

### Ideas
- **XP system**: earn points for completing tasks, capturing notes, maintaining streaks
- **Streak tracking**: daily capture streak, focus session streak
- **Achievement badges**: "First voice note", "7-day streak", "100 tasks done", etc.
- **Level system**: "You're a Level 12 Brain Architect" — give the grind a narrative
- **Daily quest**: AI picks 3 focus tasks as today's "quest" — complete them to earn XP
- **Boss battles**: big scary tasks framed as boss fights with a health bar that goes down as you complete subtasks
- **Focus streaks**: the longer you stay in focus mode without distraction, the higher the multiplier
- **Weekly leaderboard** (opt-in): compare with friends or public ADHD community
R - Have a score cost for tasks so we can rank best tasks, that is calculated based on priority/dead line / complexity / energy cost / time to completr

### Principles
- Rewards must be immediate, not delayed
- Never punish — only reward (ADHD users already have shame)
- Celebrate "good enough" — not just perfect completion
- Progress bars > numbers (visual dopamine hits)

---

## 5. Body Doubling & Social Features [AI]

> Body doubling (working alongside others) is one of the most effective ADHD tools.

### Ideas
- **Virtual coworking rooms**: join a session, set a goal, work silently alongside others
- **Accountability partner**: pair with another Infinite Brain user for daily check-ins
- **Focus room status**: "Rian is in deep focus until 3pm" — visible to your team
- **Public "working on" status**: share what you're focused on (like Last.fm but for work)
- **Group challenges**: "This week: capture 5 voice notes" — community challenges
- **ADHD community space**: forum/feed for tips, wins, and struggles (optional, non-addictive design)

---

## 6. The "Now" Experience (Task Interface) [R needed]

> The most important screen in the app is "what do I do right now?"

### Ideas
- **Single task view**: show ONE task, full screen, nothing else
- **"Now" mode**: everything else disappears, just the task and a timer
- **Guided task mode**: show micro-steps one at a time, tap to advance
- **Distraction panic button**: "I'm getting distracted" — logs the distraction, gently returns to task
- **Task soundtrack**: ambient noise or lo-fi music autoplay when entering focus mode
- **Countdown to deadline**: for tasks with due dates, show "12 hours left" prominently
- **Resistance indicator**: user rates how hard it feels to start this task (1-5) → AI learns patterns

---

## 7. Review & Reflection System

### Already Planned
- Daily digest — T-025
- Weekly review — T-026

### New Ideas [AI]
- **Monthly patterns**: what types of tasks get consistently delayed? What projects are you avoiding?
- **Life areas balance**: are you neglecting health? Relationships? Creative work?
- **"Wins wall"**: a visual gallery of completed projects and major tasks — fight recency bias
- **Retrospective prompts**: AI asks 3 questions after finishing a project ("What worked? What didn't? What would you do differently?")
- **Energy/productivity correlation**: show which days/times you're most productive based on actual data
- **Capture quality review**: "You captured 34 things this week. 12 were processed. 22 are in Inbox — want to triage them now?"

---

## 8. Integration Ideas (New/Extended)

### Developer-Specific [AI]
- **VS Code extension**: capture a TODO comment → syncs to Infinite Brain as a task
- **GitHub Issues** sync: issues assigned to you appear as tasks
- **Linear** integration: sync sprints and issues
- **Terminal capture**: `ibrain capture "remember to check the auth flow"` CLI command

### Life OS Integrations [AI]
- **Apple Health**: correlate sleep/steps/HRV with productivity patterns
- **Spotify**: log what you listened to during focus sessions (music patterns for productivity)
- **Kindle/Apple Books**: import highlights and passages
- **Duolingo / learning apps**: track learning habits alongside work habits

### Communication [existing + new]
- **Discord**: save a message to Infinite Brain directly from Discord
- **Notion**: two-way sync for shared team notes
- **Obsidian plugin**: use Infinite Brain as the sync backend for Obsidian vaults

---

## 9. Mobile & Watch Experience

### Already Planned
- Apple Watch notifications — T-044
- Task switch haptics — T-045

### New Ideas
- **Complications on watch face**: current task name + remaining time
- **Watch standalone capture**: voice note without phone
- **iOS Widget**: "What's next" widget on home/lock screen, tap to start task
- **Siri shortcut**: "Hey Siri, add to Infinite Brain: [thought]"
- **Action button (iPhone 15+)**: map to "quick capture" or "start focus mode"
- **Dynamic Island**: show current task and timer in Dynamic Island (iPhone 14 Pro+)
- **StandBy mode**: full-screen focus timer and current task when iPhone is charging

---

## 10. Business & Monetization Ideas [needs discussion]

- **Freemium**: core capture + basic tasks free, AI features + integrations paid
- **Pro tier**: unlimited AI, all integrations, body doubling rooms, advanced analytics
- **Teams tier**: shared projects, team focus rooms, manager dashboard
- **ADHD Coach marketplace**: connect users with certified ADHD coaches through the app
- **White-label API**: sell the ADHD workflow engine to other productivity apps
- **Enterprise**: for companies with neurodivergent-inclusive HR policies

---

## 11. Design Philosophy Ideas

- **Calm by default**: no red badges, no guilt-inducing overdue counts — use warm colors and soft nudges
- **Progressive disclosure**: simple on day 1, reveals depth as user grows
- **Anti-shame design**: never show "X days missed", show "resume your streak anytime"
- **Sensory-friendly options**: dark mode, reduced motion, adjustable font sizes
- **One screen, one job**: each screen does one thing; avoid feature-dense dashboards
- **Celebration animations**: dopamine-rewarding micro-animations for task completion (but skippable)

---

## 12. Open Questions & Things to Explore

- [ ] Should the Journal be a core feature or a separate "area" type? [R to decide]
- [ ] How do we handle the privacy of AI coaching conversations?
- [ ] Is gamification opt-in or always-on?
- [ ] What does "offline-first" look like for this app?
- [ ] Should we have a web app or mobile-only at launch?
- [ ] How do we avoid becoming "yet another Notion graveyard"? What keeps users coming back?
- [ ] Is body doubling a community feature or an AI simulation?
- [ ] How deep does the Apple Health integration go? Do we need HealthKit permissions?
- [ ] What's the minimum loveable product (MLP) — what's the smallest version that ADHD users would pay for?

---

## 13. Competitive Landscape Notes

| App | What They Do Well | Our Advantage |
|---|---|---|
| Notion | Flexibility, power | Too complex for ADHD; we're opinionated and AI-first |
| Obsidian | Local-first, linking | Not ADHD-specific; no proactive AI |
| Things 3 | Beautiful UX, task management | No second brain, no AI, no capture engine |
| Todoist | Cross-platform, simple | No context, no knowledge management |
| Readwise | Surfacing forgotten knowledge | Passive; no task/workflow engine |
| Inflow / ADHD apps | ADHD-specific content | Mostly educational, not a productivity system |
| Lunatask | Mood + tasks | Less AI, less capture breadth |
| Roam Research | Networked thought | Too nerdy, no ADHD workflow features |

---

## 14. Global Research — What Different Cultures Teach Us About ADHD

> Web research across 6 languages and cultures. Each region has a distinct angle we can steal from.

---

### 🇧🇷 Brazil — Emotional Warmth & Reducing Shame

**Cultural context**: Brazil has a growing neurodivergent community and strong ABDA (Associação Brasileira do Déficit de Atenção). A critical insight from Brazilian clinicians: many therapeutic approaches come from Western/North American contexts and **don't account for local cultural, social and economic specificities**. Brazilian practitioners push for culturally-sensitive interventions.

**What's working:**
- **FocoIntent** — Brazilian app launched 2025, created by a dev with ADHD. Philosophy: "restriction is power" — the app deliberately removes features. No dashboards, no tutorials. Just type and create. One text field = zero friction. When the Pomodoro ends, a visual celebration fires. [Source](https://modointent.com/focointent-gestao-de-tarefas-tdah/)
- **Focus TDAH** — Brazilian app on App Store with Pomodoro + task focus
- **Portal Neurodivergente** — psychoeducation platform built by a psychiatrist + psychologist; normalizes ADHD with warm, non-clinical language
- Binaurais (binaural beats) playlists are widely used for sensory regulation in Brazilian ADHD communities — treat this as a legit feature request
- Strong emphasis on "learn to say NO" — impulsivity makes Brazilians with ADHD over-commit; the app could track commitments and warn before adding more

**Ideas for us:**
- Anti-overcommitment guard: "You already have 7 active tasks. Are you sure?" [hot]
- Binaural/ambient audio integration built-in (not just white noise)
- Psychoeducation micro-content embedded in the app — not just a tool but a teacher
- Warm, human language in all UX copy — never clinical, never shame-inducing

---

### 🇨🇳 China — Neuroscience + Digital Medicine

**Cultural context**: China has taken a medical/neuroscience-first approach. In 2023, a Chinese company received China's **first "electronic prescription drug"** approval for ADHD — a digital cognitive training software for children 6-12. This is unprecedented globally.

**What's working:**
- **IBT Infinite Brain Technology** (Beijing) — builds ADHD digital therapy using serious games, AI, VR, biofeedback, and somatosensory technology. Research base: Beijing Normal University's State Key Laboratory of Cognitive Neuroscience. [Source](https://www.wjbrain.com/en/about)
- **AET (ADHD Executive Function Training)** — a battery of digital training tasks adapted from N-back tasks, visual-spatial memory, Schulte Grid, Go/No-go, and mental calculation. Difficulty auto-adjusts to the user's skill level — essentially adaptive difficulty like a game
- Computerized **working memory training** has shown significant improvements in Chinese children with ADHD + reading disorders (BMC Psychology, 2024)
- "AI + big data" personalization: create customized intervention plans with individual difficulty levels

**Ideas for us:**
- **Schulte Grid** mini-game as a focus warm-up before starting deep work [hot]
- **Adaptive difficulty**: tasks and focus sessions that get harder as the user improves
- **Working memory exercises** as a 2-min daily brain warm-up
- **N-back tasks** as an optional daily "brain training" feature
- Biofeedback integration (HRV, heart rate from Apple Watch) to detect stress before it kills focus

---

### 🇯🇵 Japan — Kaizen, Visual Systems & "Mitooshi"

**Cultural context**: Japan has a unique lens — the concept of **mitooshi (見通し)** meaning "foresight" or "being able to see ahead". Japanese ADHD research shows that people with ADHD struggle specifically with mitooshi — they can't mentally simulate the future steps of a task. This is a deeper description of task initiation than just "dopamine".

**What's working:**
- **Conductor (コンダクター)** — Japanese app specifically built for developmental disabilities. Core insight: "show what's coming next". Visual timeline that shows remaining time and future steps. Philosophy: build *mitooshi* (foresight) artificially through UX. [Source](https://co-coco.jp/news/conductor/)
- **Kaizen System** (Notion-based, Japanese-inspired) — combines three pillars: Brain Dump (clear mental clutter), Daily Highlight (one task for success), Micro-commitment (2-min habit builder)
- **Ikigai** as a motivational filter — Japanese ADHD practitioners ask "is this task connected to your ikigai?" If not, can it be delegated or dropped?
- **Shinrin-yoku (forest bathing)** is used as a prescribed reset for hyperfocus burnout
- **Kanban** boards are deeply embedded in Japanese productivity culture — physical visual systems externalize what the brain can't hold

**Ideas for us:**
- **Mitooshi mode**: before starting a task, AI shows you the next 3 steps so your brain can "see ahead" — reduces initiation paralysis [hot]
- **Kanban view** as a first-class alternative to list view
- **Ikigai alignment score**: when adding a task, optionally tag if it connects to your purpose/goals — AI surfaces this in priority scoring
- **2-minute micro-commitment**: every day the app asks for just one 2-min habit. No more. Builds momentum without overwhelm.
- "Brain warm-up" ritual before work (structured, short, repeatable)

---

### 🇪🇸🌎 Spanish-Speaking World — Single-Task Focus & Gamification

**Cultural context**: Spain and Latin America share the TDAH framing. The Spanish-speaking ADHD community has produced some interesting apps and strong awareness content.

**What's working:**
- **Addie (TDAH y Productividad)** — built by people with ADHD, backed by ADHD Foundation. Key features: [Source](https://apps.apple.com/cr/app/addie-tdah-y-productividad/id6444955141)
  - Shows **one task at a time** — swipe card interface
  - Gamification engine: rewards for completing tasks ("keep dopamine flowing")
  - Prioritizes by urgency + importance + **fun** — fun is a first-class priority criterion
  - Tracks how long tasks actually take vs. estimated (time blindness training)
  - Tracks mood, medication, menstrual cycle — looks for correlations with productivity
  - Built-in pros/cons list for decisions (reduces decision paralysis)
- **CogniFit** (Spanish company) — multi-dimensional cognitive training for ADHD adults

**Ideas for us:**
- **Fun as a task attribute**: let users rate how fun a task is. AI uses this in scheduling (don't stack boring tasks). [hot]
- **Pros/cons mini-tool** for decisions (inside the app, not a separate app)
- **Medication + mood + productivity correlation** tracking — show the user their own patterns
- **Time reality tracker**: "You estimated 30 min. It took 1h 40min. Want to adjust your estimate for next time?" — trains time awareness over months

---

### 🇫🇮 Finland & Nordic Countries — Universal Design & Systems Thinking

**Cultural context**: Finland has arguably the world's most advanced neurodivergent-inclusive education system. The Finnish Education Act **mandates** that all students learn in mainstream classrooms regardless of learning needs. This "Universal Design for Learning" (UDL) philosophy is deeply embedded.

**Key insights:**
- Finland uses **three-tier support**: general, intensified, special. The idea: help before a crisis, not after
- Finnish classrooms use **flexible lesson structures** — shorter segments + movement breaks built in, not added on top. The app should do the same: make breaks structural, not optional
- **Individualized learning plans** written collaboratively by teacher + parent + child. For our app: onboarding that builds a personal ADHD profile with the user, not just a generic setup
- In 2021 Finland added ADHD/neurodivergent competence as a **required skill for all teachers** — this is cultural normalization at scale
- Nordic workplaces: **flexible hours, quiet rooms, noise-canceling headphones, movement breaks, remote work** are not accommodations — they're defaults. ADHD users perform 30% better in these environments.
- Research finding: teams with neurodiverse professionals showed **30% higher productivity and 90% retention** compared to neurotypical-only teams

**Ideas for us:**
- **Onboarding as a personal ADHD profile builder**: ask about energy patterns, peak hours, triggers, preferred work style — tailor the entire app to this profile [hot]
- **Structural breaks**: breaks are part of the schedule, not a reward for finishing. Infinite Brain treats breaks as mandatory.
- **Quiet mode / sensory settings**: reduced animations, no sound, monochrome option — not an afterthought, a core feature
- **"Accommodate first" philosophy**: don't wait for the user to be struggling. Default settings should already be ADHD-optimized.
- **Team mode** (future): manager dashboard that shows team neurodiversity profile — help managers support ADHD employees

---

### 🕌 Arab World — Spirituality, Structure & the Saudi ADHD Society

**Cultural context**: Saudi Arabia has a dedicated NGO, the **Saudi ADHD Society**, founded in 2004 — the first in the Arab world. The Arab world brings a unique angle: the intersection of ADHD management with Islamic practice and community accountability.

**What's working:**
- **Saudi ADHD Society** runs Life & Career Management workshops covering time management tools and productivity strategies, and has MOU with Ministry of Education for ADHD teacher training. [Source](https://adhd.org.sa/en/)
- **Salah (prayer) as a natural time-boxing system**: 5 daily prayers create fixed time anchors throughout the day. For Muslim users with ADHD, Salah is a built-in Pomodoro — the day is already structured around 5 checkpoints. [Source](https://hayatplanners.co.uk/blogs/the-blog/enhancing-productivity-with-adhd-remembering-salah-as-a-muslim)
- **Community accountability** is core — Arab culture is collectivist. Shame and accountability work differently; peer support groups (not individual apps) have high adoption
- **Religiously-sensitive mindfulness** — standard Western mindfulness apps (e.g., Headspace) don't resonate. Mindfulness framed through Islamic concepts (tawakkul, tafakkur) has better adherence in Arab populations
- **ArabTherapy** platform provides ADHD assessment tools in Arabic

**Ideas for us:**
- **Custom daily anchor points**: not just Pomodoro — let users define their own daily structure anchors (prayer times, meals, school pickup, medication). AI respects these as hard boundaries. [hot]
- **Cultural/religious calendar integration**: Islamic Hijri calendar, Ramadan mode (schedule adapts during fasting), Jewish Shabbat mode, etc.
- **Community accountability rooms** (body doubling variant) — work with people you trust, not strangers
- **Localization as a first-class feature**: app should feel native in Arabic (RTL), not translated. Same for Portuguese (Brazilian), Japanese, Chinese.
- **Offline-first** is critical in markets with inconsistent connectivity (LATAM, MENA)

---

### Cross-Cultural Synthesis — Universal Insights

| Region | Unique Contribution | Feature Idea for Infinite Brain |
|---|---|---|
| 🇧🇷 Brazil | Shame reduction, warmth, anti-overcommitment | "Too many tasks" guard, binaural audio, psychoeducation bites |
| 🇨🇳 China | Neuroscience games, adaptive difficulty, biofeedback | Schulte Grid warm-up, N-back, HRV from Apple Watch |
| 🇯🇵 Japan | Mitooshi (foresight), Kaizen, visual systems | "See ahead" mode, Kanban, Ikigai alignment |
| 🇪🇸 Spain/LATAM | Fun as a task attribute, medication tracking | Fun score, time reality tracker, pros/cons mini-tool |
| 🇫🇮 Finland | Universal Design, structural breaks, personalized onboarding | ADHD profile builder, mandatory breaks, sensory settings |
| 🕌 Arab World | Prayer as time-boxing, community accountability, cultural calendar | Custom anchors, Ramadan mode, RTL support, community rooms |

**The big insight**: every culture independently discovered that ADHD needs **external structure + community + immediate reward**. No one discovered a magic productivity hack that's culture-specific. The universal formula is: **zero friction capture + visible time + one thing at a time + immediate dopamine + trusted accountability**.

---

## 15. Apps Worth Studying (Global Competitive Map)

| App | Country | Key Differentiator | What We Can Learn |
|---|---|---|---|
| FocoIntent | 🇧🇷 Brazil | One text field, extreme simplicity | Restriction is a feature; less is more |
| Addie | 🇪🇸 Spain | Fun as a task attribute, swipe cards | Fun-first prioritization |
| Conductor (コンダクター) | 🇯🇵 Japan | Mitooshi — "see ahead" visual timeline | Show next steps before starting |
| IBT / AET | 🇨🇳 China | FDA/NMPA-level brain training games | Cognitive warm-up before work |
| Tiimo | 🇩🇰 Denmark | Visual planner, built by/for neurodivergent | Visual time, not lists |
| FLOWN | 🇬🇧 UK | Body doubling, async coworking | Social focus sessions |
| Focusmate | 🇺🇸 USA | Live body doubling with strangers | Accountability pairing |
| Lunatask | 🇸🇰 Slovakia | Mood + tasks + journal in one | Integrated wellness + productivity |
| Forest | 🇨🇳 China | Gamified focus (plant a tree) | Visual growing reward |
| Inflow | 🇺🇸 USA | ADHD CBT program inside an app | Psychoeducation built-in |
| Brili | 🇺🇸 USA | Visual routine builder, time blindness | Drag-and-drop time-aware routines |

---

## Research Sources

### English
- [12 Best Productivity Apps for ADHD in 2025 — Fluidwave](https://fluidwave.com/blog/productivity-apps-for-adhd)
- [7 Best ADHD Productivity Apps for Focus & Planning in 2026 — Morgen](https://www.morgen.so/blog-posts/adhd-productivity-apps)
- [Second Brain Strategies for ADHD Users That Actually Stick — Medium](https://medium.com/@theo-james/second-brain-strategies-for-adhd-users-that-actually-stick-83a785290a08)
- [Popular Productivity Advice That Works Against the ADHD Brain — ADDitude](https://www.additudemag.com/adhd-brain-productivity-advice/)
- [10 ADHD-Friendly Productivity Strategies That Actually Work — Calm a Lama](https://www.calmalama.com/blog/adhd-productivity-strategies)
- [Body Doubling for ADHD — FLOWN](https://flown.com/body-doubling-for-adhd)
- [ADHD Gamification — ADHD Centre](https://www.adhdcentre.co.uk/adhd-gamification-and-its-role-in-boosting-focus-and-learning/)
- [Gamified Task Management for ADHD — MagicTask](https://magictask.io/blog/gamified-task-management-adhd-focus-productivity/)
- [Task Initiation Tactics for ADHD Adults — Tiimo](https://www.tiimoapp.com/resource-hub/task-initiation-adhd)
- [Toward Neurodivergent-Aware Productivity (AI + ADHD research paper) — arXiv](https://arxiv.org/html/2507.06864)
- [Best ADHD Planner and Productivity App — Lunatask](https://lunatask.app/adhd)
- [Inclusive Education in Finland — TechClass](https://www.techclass.com/resources/education-insights/finlands-approach-to-special-education-how-every-student-gets-individualized-support)
- [IBT Infinite Brain Technology — About](https://www.wjbrain.com/en/about)
- [Improving cognitive function in Chinese children with ADHD — BMC Psychology](https://bmcpsychology.biomedcentral.com/articles/10.1186/s40359-024-02065-1)
- [Saudi ADHD Society](https://adhd.org.sa/en/)
- [Enhancing Productivity with ADHD: Remembering Salah as a Muslim — Hayat Planners](https://hayatplanners.co.uk/blogs/the-blog/enhancing-productivity-with-adhd-remembering-salah-as-a-muslim)
- [Thriving with ADHD in the Muslim Community — Centre for Muslim Wellbeing](https://cmw.org.au/2024/05/23/thriving-with-adhd-in-the-muslim-community-challenges-and-support/)

### Português (Brasil)
- [FocoIntent — Gestão de Tarefas para TDAH lançada em 2025](https://modointent.com/focointent-gestao-de-tarefas-tdah/)
- [TDAH no Adulto — estratégias para o dia a dia — ABDA](https://tdah.org.br/tdah-no-adulto-algumas-estrategias-para-o-dia-a-dia/)
- [Melhores Apps para TDAH — Anayara Fraga](https://www.anayarafraga.com/post/melhores-app-para-tdah)
- [Portal Neurodivergente](https://www.portalneurodivergente.com.br/)
- [Os 5 melhores apps de rotina e foco para adultos com TDAH em 2025](https://tryhero.app/articles/portuguese-articles/os-5-melhores-apps-de-rotina-e-foco-para-adultos-com-tdah-em-2025)

### Español
- [Addie — TDAH y Productividad (App Store)](https://apps.apple.com/cr/app/addie-tdah-y-productividad/id6444955141)
- [Método del Segundo Cerebro — Formas Formación](https://formasformacion.com/en-que-consiste-el-metodo-del-segundo-cerebro-y-por-que-ayuda-a-nuestra-productividad/)
- [TDAH en adultos — CogniFit](https://www.cognifit.com/es/tdah-adultos)
- [¿Qué Apps para TDAH Adulto Realmente Funcionan en 2025?](https://tryhero.app/articles/spanish-articles/qu%C3%A9-apps-para-tdah-adulto-realmente-funcionan-en-2025)

### 日本語 (Japanese)
- [タスク管理アプリ「コンダクター」— こここ](https://co-coco.jp/news/conductor/)
- [大人のADHDとは — 武田薬品工業](https://www.otona-hattatsu-navi.jp/)
- [ADHD（注意欠如多動症）のある方の時間管理やスケジュール管理](https://works.litalico.jp/column/developmental_disorder/021/)
- [Japanese Productivity Methodologies — Wherever Magazine](https://www.wherevermags.com/news/a-guide-to-japanese-productivity-methodologies/)

### 中文 (Chinese)
- [ADHD数字疗法前沿 — IBT无疆科技](https://www.wjbrain.com/influence_detail/219)
- [AI如何升级你的"第二大脑" — CSDN](https://blog.csdn.net/2401_84587944/article/details/138166916)
- [Clinical study on digital therapy for ADHD — Scientific Reports / Nature](https://www.nature.com/articles/s41598-024-73934-3)

### العربية (Arabic)
- [Saudi ADHD Society — جمعية إشراق](https://adhd.org.sa/en/)
- [اختبار ADHD — عرب ثيرابي](https://arabtherapy.com/ar/tools/adhd)
- [MOH Protocol for ADHD — Saudi Ministry of Health](https://www.moh.gov.sa/en/Ministry/MediaCenter/Publications/Documents/MOH-Protocol-for-ADHD-Across-the-Life-Span.pdf)

---

---

## 16. Session Brainstorm — 2026-04-01 [R + AI]

### Core Vision Refinement [R]

Infinite Brain is not just an ADHD capture tool — it is a **full life OS** and a **logic graph for a software engineer**. Both in the same brain. Drop a movie, a doctor appointment, a business rule, a code decision — all captured through the same inbox, routed by AI to the right place.

As a software engineer, work requires transforming business rules into logic. The brain needs to hold logics from many different projects simultaneously. AI must be able to fast-retrieve the relevant context, persist memory across sessions, and parallel agents can share the same memory to update things in real time.

---

### Knowledge Graph Layer [AI] `[hot]` → T-028

A graph layer that sits on top of all content (notes, tasks, ideas, rules). Nodes are any entity — idea, business rule, code decision, movie, doctor appointment, book. Edges are reasoned relationships.

**Node schema:**
```
nodes (id, type, title, content, para, project_id, embedding vector(1536), metadata jsonb)
```
- `type`: note | task | event | media | rule | insight | decision | ...
- `para`: project | area | resource | archive
- `metadata jsonb`: flexible per type — movie gets `{ genre, year }`, rule gets `{ project, language }`, appointment gets `{ scheduled_at, location }`
- `embedding`: pgvector for semantic proximity

**Edge schema:**
```
edges (from_node, to_node, relation_type, confidence, created_by)
```
- `relation_type`: implements | solves | contradicts | relates | inspired_by | blocks

Explicit edges = reasoned relationships. pgvector = semantic proximity. Both needed.

---

### AI Session Memory [AI] `[hot]` → T-016

When a Claude session ends, what was reasoned should not die. A new table stores AI-generated observations during a session:

```
agent_memories (id, session_id, agent_id, content, type, embedding, project_id, created_at)
```

- User notes = captured content
- Agent memories = AI reasoning traces
- Next session loads last N relevant memories for current context
- Parallel agents read/write the same table scoped to a project — the database is the shared memory bus

---

### Cross-Project Insight Linker [AI] `[hot]` → T-029

Nightly Asynq cron job that mines the knowledge graph for cross-project connections:

1. Query all nodes across all projects
2. Run cosine similarity over embeddings — look for high-similarity nodes in **different** projects
3. When similarity > threshold AND different projects AND no edge exists → candidate insight
4. Create a new node of type `insight` with edges to both source nodes
5. Surface in next daily digest

Example: a solution pattern in Project A that solves a problem in Project B. Or a book you read that maps to a technical decision you made. The brain gets smarter over time, not just bigger.

---

### Relevance Decay + Review Ladder [R] `[hot]` → T-036

Every node has a `review_stage` (0–5) and `next_review_at`. Items that go unanswered keep surfacing. Items that receive "Yes" move to confirmed biannual cadence. Four consecutive "No" answers over ~16 months lead to deletion.

**Stage table:**

| Stage | Label | Interval | "Yes" | "No" | Silence |
|---|---|---|---|---|---|
| 0 | pending | bi-weekly | → confirmed | → doubt_1 | resurface same |
| 1 | confirmed | 6 months | stay | → doubt_1 | resurface same |
| 2 | doubt_1 | 1 month | → confirmed | → doubt_2 | resurface same |
| 3 | doubt_2 | 3 months | → confirmed | → doubt_3 | resurface same |
| 4 | doubt_3 | 6 months | → confirmed | → doubt_4 | resurface same |
| 5 | doubt_4 | 6 months | → confirmed | **deleted** | resurface same |

Rules:
- Silence = no change, item resurfaces at the same interval next cycle (no silent advancement)
- "Yes" at any stage = jump to confirmed (stage 1, biannual)
- "No" at stage 5 = hard delete (total ~16 months from first "No")
- Stage 5 → "No" is the only delete trigger — no grace buffer

---

### Daily Chunk Planner [R] `[hot]` → T-048

Replace/extend T-040 (focus timer) and T-047 (daily planner) with a structured chunk-based day planner built for ADHD.

**Core idea:** the day has N chunks (default 16). Each chunk has a type (work, chore, exercise, personal, free). Order does not matter — you pick what to do next based on current energy. The system enforces 100% focus during a chunk and notifies you when it ends.

**Flow:**
```
Morning → confirm chunk mix for today (or accept template)
    └── 16 slots: [work x8][chore x3][exercise x1][personal x2][free x2]

Pick any chunk → timer starts → full focus
Timer ends → notification: "chunk done, pick next"
Repeat until 16/16 complete
```

**Key decisions:**
- When picking a **work chunk**: system asks "what are you working on?" at that moment (not pre-planned). AI suggests top 3 from task list based on priority + current energy. User picks, types custom, or skips.
- Chunk duration: customizable per template (default 60 min), can be 30 min, 90 min, etc.
- Individual chunks can have custom durations within a plan

**Schema:**
```sql
chunk_templates (id, user_id, name, chunk_min, slots jsonb)
-- slots: [{"type":"work","count":8},{"type":"chore","count":3},...]

daily_plans (id, user_id, template_id, date, chunk_min, status)
-- status: active | completed | abandoned

chunks (id, plan_id, type, task_id, duration_min, status, started_at, completed_at, sequence)
-- status: pending | active | completed | skipped
-- sequence: order in which it was actually executed (not planned)
```

**AI role at chunk-start:**
- Suggests which task to work on based on priority, deadlines, and what was done in previous chunks today
- Learns patterns over time (e.g., you prefer exercise chunks in the morning)
- Warns if you're stacking too many work chunks back-to-back without a break

---

### Open Questions from This Session

- [ ] Should the knowledge graph have a UI (visual graph view) or stay API-only at MVP? [R to decide]
- [ ] How does the insight linker surface results — daily digest only, or a separate "insights" feed?
- [ ] Chunk planner: what happens when you don't finish 16/16 by end of day? Roll over? Archive incomplete?
- [ ] Relevance decay: should the AI be able to auto-confirm items it's confident about (e.g., a recurring task pattern)? Or is confirmation always human?

---

*Last updated: 2026-04-01 — Rian + Claude (vision refinement, knowledge graph, review ladder, chunk planner)*
