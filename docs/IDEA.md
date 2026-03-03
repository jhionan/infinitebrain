# Infinite Brain

## Vision

**Entropy is the default. Infinite Brain is the energy you inject to keep things in balance.**

Look at how nature organizes itself. Protons and electrons are unstable alone — they bond
into atoms to reach a lower energy state. Atoms are unstable alone — they bond into molecules.
Molecules into cells. Cells into organs. Organs into organisms. Organisms into families.
Families into communities. Communities into nations. Nations into trade blocs. The planet
into a solar system. The pattern repeats without end, at every scale.

Two things are true at every level:
1. **Elements seek balance** — isolated elements are chaotic; they aggregate to become stable
2. **Balance requires energy** — without active input, entropy wins; bonds break; chaos returns

The same is true for knowledge. A thought in your head is unstable — it will be forgotten.
A decision made in a meeting is unstable — it will be lost. A team's collective understanding
is unstable — it disperses when people leave, change roles, or get pulled in new directions.
At every scale of human organization, the default state is entropy.

Infinite Brain is the energy injection that fights entropy at any level.

A single person uses it to keep their thoughts coherent — nothing lost, decisions recoverable,
context always restorable. A team uses it to keep their collective understanding coherent —
no repeated questions, no re-solved problems, no knowledge leaving with people. An organization
uses it to keep its intelligence coherent — seeing across silos, detecting drift, surfacing
what nobody can see alone.

The structure is open. You name your levels. You define your hierarchy. IB doesn't impose
a fixed org chart — it works with whatever structure you have, and applies the same recursive
brain model at every node in the tree, infinitely deep.

**The core promise: coherence at any scale, against the natural tendency toward chaos.**

---

## The Problem We Solve: Entropy at Every Scale

Knowledge is the most valuable asset in any human system. It is also the most entropic.
Left without energy input, it disperses, degrades, and disappears.

**At the individual level (a cell in isolation):**
- Great ideas die in the friction of "I'll write it down later"
- Decisions made months ago are forgotten when they become relevant again
- Context collapses between sessions — "where was I?"
- The brain is used as working memory: fragile, lossy, exhausting

**At the team level (cells forming an organ):**
- Teams re-solve problems that were already solved by someone who left
- Decisions live in meeting notes nobody reads or in the head of the person who was there
- Knowledge leaves when people leave — the organ loses function
- Cross-team connections never form: sales pain never reaches engineering

**At the organizational level (organs forming an organism):**
- Leadership can't see what's actually happening below the surface
- Metrics tracked are the measurable ones, not the meaningful ones
- The gap between declared values and actual behavior widens silently
- The org grows and becomes less coherent, not more — scale amplifies entropy

**At any level above (ecosystems, industries, networks):**
- The same pattern repeats
- The energy requirement for coherence grows with complexity
- Without it: fragmentation, redundancy, misalignment, collapse

Infinite Brain applies the same counter-entropic engine at every level of this hierarchy,
using the same architecture, with the same data model, infinitely composable.

---

## The Recursive Brain Model

The same four-operation brain at every level of any hierarchy, infinitely deep.

```
[any named level]
    captures → organizes → understands → acts
        ↓  (knowledge flows upward, with permission, de-attributed)
[next level up]
    captures → organizes → understands → acts
        ↓
[next level up]
    ...
```

At every level, the brain does four things:
1. **Capture** — receive knowledge from the entities at this level and from below
2. **Organize** — structure it, relate it, classify it, connect it
3. **Understand** — surface patterns, gaps, coherence state, needle-movers
4. **Act** — generate tasks, dispatch agents, surface decisions, inject energy

**The levels are not fixed.** You define the hierarchy. A startup has two levels.
A multinational has ten. A community org has its own structure. The engine doesn't care.
The `org_unit` table is a free-form tree: any name, any depth, no predefined types required.

**The data model is the same at every level** — nodes, edges, visibility scoped to the unit.
A team brain is a filtered view of the knowledge graph scoped to that unit.
A company brain is the full graph with anonymized aggregation at the top.
Each level is coherent independently; each level also contributes to the level above.

**Coherence is the metric.** At every level, IB measures and surfaces:
- How much knowledge is captured vs. how much is probably missing (knowledge density)
- How well-connected that knowledge is (graph density)
- How aligned current work is with the declared True North (alignment)
- Whether knowledge is distributed or concentrated in single points of failure
- How fresh the knowledge is — decaying context is a coherence risk

When coherence is low, IB injects energy: surfaces gaps, connects isolated nodes,
flags concentration risks, suggests what the level needs to capture to restore balance.

---

## Core Principles

### 1. One Source of Truth

There is one knowledge graph. Not Notion for docs, Jira for tasks, Slack for decisions, and
someone's head for the actual context. One graph. Every entity is a node. Every relationship
is an edge. The whole truth lives here.

This is non-negotiable. Tools that silo knowledge are the problem. IB is the solution.

### 2. Capture First, Organize Later

Every input channel is zero-friction. Voice note while driving. Forward an email. Message
the bot. Screenshot a whiteboard. Everything lands in the Inbox — unclassified, unstructured,
exactly as it arrived. AI handles classification, tagging, and routing.

The capture step must have zero friction. If it takes more than 5 seconds, thoughts are lost.

### 3. True North — The Alignment Anchor

Before IB can help you act with clarity, it needs to know what you're actually optimizing for.
Not the polished mission statement — the honest one.

A person's True North: "I want to be engineering lead within 2 years without working more than
45 hours a week."

A company's True North: "We need to be profitable in 14 months. Retention matters more than
acquisition right now. We won't grow past 12 people."

True North is the root node of the knowledge graph. Every task, project, idea, and decision
has an alignment score relative to it. IB prioritizes by alignment, surfaces drift when
decisions consistently contradict the stated True North, and gives AI agents a real objective
function when decomposing work.

**Most tools track what you do. IB tracks whether it's moving you toward what you actually want.**

### 4. Privacy by Architecture — Not by Policy

The recursive model only works if people trust it enough to contribute. That trust requires
different privacy contracts at each level.

**Individual**: Everything is yours. Nothing leaves your personal brain without your explicit action.
Full attribution. You are data subject and data controller.

**Team**: Small trusted group. You explicitly publish to the team. Attribution is visible to teammates —
this is a safe, known circle.

**Higher levels (squad → unit → org)**: Knowledge contributes to collective intelligence, but the
individual is technically removed from the output. IB sees everything. The org sees patterns, not people.

The escalation from personal to org is **irreversible and always explicit**. The system never
auto-publishes. Privacy enforcement is in the architecture — not policies that can be bypassed.

### 5. Intelligence Without Surveillance

The org brain's value depends entirely on people contributing to it. People only contribute
if they feel safe. They only feel safe if they know the system cannot be used against them.

> **We measure the health of the organization's collective knowledge.
> We never benchmark individuals.**

Technical enforcement (not just policy):
- Org-level queries are blocked if fewer than 5 people contributed to the result (k-anonymity)
- Individual attribution is stripped at ingestion for org-level contributions
- There is no API endpoint for "what did [person] contribute"
- There are no leaderboards, contribution scores, or individual activity metrics
- These constraints are enforced at the service layer and are not configurable

This is also a legal requirement (EU AI Act — see T-154, T-157).

### 6. AI as Active Collaborator — Aligned to True North

AI doesn't just file notes. It:
- Surfaces forgotten context when you start a new task
- Connects knowledge across the org that would never meet otherwise
- Detects drift from True North before it becomes a crisis
- Generates tasks from business rules and acceptance criteria
- Dispatches agents to execute work and verifies completion

Every AI decision is labeled (T-155), explained on request (T-156), and overridable by the human.
AI is the intelligence layer — not the decision maker.

### 7. The Productivity Layer — Inherited from ADHD Research

The individual brain layer is built on ADHD research — not because IB is an ADHD tool,
but because ADHD reveals the sharpest version of universal cognitive challenges:

- **Time blindness** → visible countdown timers, gap tracking ("you said 30 min, it's been 1h 12m")
- **Task initiation** → micro-first-steps, "just 5 minutes" mode, frictionless start
- **Decision fatigue** → "What do I do now?" button, AI-curated daily top 3
- **Context loss** → "Where was I?" session restoration, breadcrumb trail
- **Hyperfocus** → gentle nudges when a session runs beyond threshold

These patterns benefit every knowledge worker, not just those with ADHD. The productivity
layer is a competitive advantage — it's just no longer the product's identity.

### 8. The Meta-Loop — IB Builds IB

Infinite Brain is the first test of its own architecture. The `features/` specs, `docs/TASKS.md`,
`docs/ARCHITECTURE.md`, every decision in this project — these are nodes in IB's knowledge graph.

When we need to implement a feature:
1. The spec is already in IB
2. IB queries relevant business rules and architectural constraints
3. IB generates acceptance tests from the criteria
4. IB dispatches an agent with full context
5. The agent implements; the tests verify; IB marks it complete

The product proves itself by building itself. This is the portfolio demo and the
development methodology at the same time.

---

## The Open Hierarchy — Structure Emerges From Reality

IB does not impose an organizational structure. It mirrors yours.

The `org_unit` tree is free-form. You name your levels. You define depth.
A startup might have:
```
Company
└── You
```

A scale-up:
```
Company
├── Engineering
│   ├── Platform Team
│   └── Product Team
└── Go-to-Market
    ├── Sales Pod
    └── Marketing Pod
```

A global organization:
```
Parent Org
├── Region: Americas
│   ├── Country: Brazil
│   │   ├── Business Unit: Retail
│   │   └── Business Unit: B2B
│   └── Country: USA
└── Region: Europe
    └── ...
```

A research community:
```
Network
├── Working Group: Privacy
├── Working Group: AI Governance
└── Collaborative Project: X
    ├── Sub-team: Research
    └── Sub-team: Implementation
```

All of these use the same `org_units` table with `parent_unit_id` as a self-referencing tree.
The brain model applies at every node in the tree. The levels have no predefined meaning.
The structure is whatever helps the people inside it maintain coherence.

**This mirrors how nature works**: the hierarchy emerges from the need for balance,
not from a predefined taxonomy. Cells don't know they're inside an organ. They just
seek equilibrium with their neighbors. The organ emerges from that behavior.
IB makes the emergent structure visible and keeps it coherent.

---

## User Personas

### The Knowledge Worker (entry point — individual tier)
- Developer, researcher, consultant, designer, writer
- Thinks for a living; manages complex projects with many moving parts
- Current state: knowledge scattered across Notion/Obsidian/Slack/their head
- Wants: one place where nothing is lost and context is always recoverable
- "I want to stop rediscovering things I already know"

### The Solo Technical Founder (power user — all tiers)
- Building a product alone or with one co-founder
- Wears all hats: product, engineering, design, sales
- Needs decisions from 6 months ago to surface when they become relevant
- Wants AI agents to execute correctly because the context is there
- "I want to describe what I want and have it done right — no hand-holding required"

### The Engineering Team Lead (team tier)
- 3–10 engineers, moving fast
- Documentation is always out of date; nobody agrees on why decisions were made
- Wants AI coding tools to have real domain context, not hallucinate
- "When a new engineer asks why we did X, I want IB to answer — not me"

### The COO / Chief of Staff (org tier)
- 50–500 person company
- Information silos are costing money: sales doesn't know what engineering knows
- People leave; knowledge leaves with them
- Can't see what actually drives revenue from the dashboard
- "I want to know what I don't know before it costs us"

### The People & Culture Lead (org tier)
- Responsible for onboarding, knowledge transfer, and organizational health
- Nightmare: one person is the single point of failure for a critical process
- Wants expertise mapped without surveying people
- "When someone leaves, I want their knowledge to stay"

---

## The Intelligence Stack

| Layer | Who uses it | What it does |
|---|---|---|
| Individual Brain | Everyone | Memory augmentation, decision support, focus, context restoration |
| Team Brain | Teams of 3–20 | Shared knowledge, expertise graph, decision continuity, onboarding |
| Unit / Org Brain | Companies | Collective intelligence, needle-mover analysis, drift detection, idea pipeline |
| Platform Layer | Dev teams | Business rules, requirements, agent orchestration, context API |

---

## Success Metrics

**Individual tier:**
- Capture a thought in under 5 seconds from any device
- Zero lost decisions — every major decision has a node with rationale
- "Where was I?" restores context in under 10 seconds
- User reports less cognitive load after 30 days

**Team tier:**
- "Has anyone solved this before?" answered by IB in under 5 seconds
- New team member self-serves answers from IB instead of interrupting colleagues
- Decision rationale retrievable for any decision made in the last 2 years

**Org tier:**
- IB surfaces at least one cross-silo insight per week that leadership acts on
- Knowledge concentration risk detected before the person leaves (not after)
- True North drift detected within 30 days of it beginning

**Platform tier:**
- AI coding tools using the Context API produce demonstrably fewer domain errors
- Agent-executed tasks pass acceptance tests on first attempt > 70% of the time

---

## Key Features by Layer

| Layer | Category | Features |
|---|---|---|
| Individual | Capture | Text, voice (Whisper), email, bots, webhooks, web clipper |
| Individual | Organization | PARA + knowledge graph, auto-classify, auto-tag, relationship graph |
| Individual | Intelligence | Semantic Q&A, daily digest, True North alignment, decision journal |
| Individual | Action | Now/Next/Later tasks, focus timer, context restoration, hyperfocus guard |
| Team | Knowledge | Shared nodes, expertise graph, decision continuity, onboarding acceleration |
| Team | Collaboration | Explicit publish, team-scoped search, shared inbox |
| Org | Intelligence | Anonymized insights, needle-mover analysis, gap detection, idea pipeline |
| Org | Health | Drift detection, knowledge risk, org health metrics |
| Platform | Dev Tools | Business rules, requirements, test generation, agent dispatch, Context API |
| Platform | Compliance | SOC2 + HIPAA + EU AI Act + GDPR controls |

---

## Monetization

### Open Core SaaS (AGPL-3.0)

The engine is open source. Revenue comes from the managed cloud and premium layers.

| Tier | Price | Includes |
|---|---|---|
| Personal | Free | 1 user, limited AI ops/month, all individual features |
| Pro | $12/mo | Unlimited AI, all integrations, team features for 1 |
| Team | $18/user/mo | Team brain, shared workspace, expertise graph (min 3 seats) |
| Org | $25/user/mo | Full org intelligence, idea pipeline, drift detection (min 20 seats) |
| Enterprise | Custom | On-premise, SLA, compliance evidence generation, dedicated support |

### Additional Revenue Streams

**Agent execution credits** — dispatch AI agents from IB; charged per run. Scales with output, not headcount.

**Context API** — dev teams pay for AI coding tools to query IB for domain context.

**Compliance evidence generation** — auto-generate SOC2/HIPAA audit packages from IB's event log.

**ADHD Coach Marketplace (post-MVP)** — certified coaches link with users; platform takes 15%.

### Open-Source Advantage

AGPL-3.0 self-hosted IB is exempt from EU AI Act high-risk obligations (Article 2(12)).
This makes the hosted cloud tier *more valuable* to enterprise buyers — compliance is handled.
Self-hosters get the tool; enterprise buyers get the managed compliance layer.

---

## What IB Is Not

- **Not a project management tool** — IB manages knowledge; tasks are a projection of that knowledge
- **Not a performance monitoring tool** — individual metrics are private; org metrics are anonymized
- **Not a replacement for judgment** — IB surfaces context and patterns; humans make decisions
- **Not another note-taking app** — notes are one node type among many; the graph is the product

---

## Inspiration & References

- Tiago Forte — *Building a Second Brain* (PARA method — the organizational backbone)
- David Allen — *Getting Things Done* (capture discipline)
- Cal Newport — *Deep Work* (focus philosophy)
- Douglas Engelbart — "Augmenting Human Intellect" (the original vision for tools that amplify thinking)
- Stafford Beer — *Brain of the Firm* (viable systems model — organizational intelligence as brain)
- Notion, Obsidian, Roam Research (existing tools with gaps — IB fills them)
- Things 3 (task UX gold standard)
- Readwise (knowledge resurfacing)
