# T-129 — Personal AI Preferences Model

## Overview

Every time a user corrects an AI decision (reclassifies a node, edits auto-tags, dismisses
an insight), that correction is a training signal. After enough corrections, a lightweight
preference model captures user-specific patterns and biases future AI decisions.

The AI gets smarter about each user over time. This is the data flywheel.

## Signal Collection

Correction signals captured via events (T-120):

| User Action | Event | Signal |
|---|---|---|
| Changes PARA category | `node.classified` with correction | Wrong classification |
| Edits auto-tags | `node.tagged` with user correction | Wrong tags |
| Dismisses insight | `insight.dismissed` | Unhelpful connection |
| Accepts insight | `insight.accepted` | Good connection |
| Changes task priority | `task.priority_scored` correction | Wrong priority |

## Preference Profile

```go
// internal/ai/preferences/profile.go

type UserPreferenceProfile struct {
    UserID uuid.UUID

    // Classification biases: user tends to classify X as Y
    // e.g. {"notes about meetings": "Project"} (despite AI saying "Resource")
    ClassificationPatterns []Pattern

    // Tag vocabulary: user's preferred tag names vs AI's
    PreferredTags   map[string]string  // AI tag → user's preferred version
    IgnoredTags     []string           // tags user always removes

    // Project vocabulary: names + abbreviations the AI doesn't know
    ProjectNames    []string
    ProjectAliases  map[string]string  // "IB" → "Infinite Brain"

    // Insight quality: what kinds of connections the user finds valuable
    InsightAcceptRate float64          // 0.0–1.0
    InsightTopics     []string         // topics where insights were accepted
}
```

## How Preferences Bias Prompts

```go
// internal/ai/prompts/classify/v1.go

func BuildPrompt(content string, prefs *UserPreferenceProfile) string {
    if len(prefs.ClassificationPatterns) > 0 {
        // Inject user-specific examples into the prompt
        // "Based on past corrections, this user classifies X as Y"
    }
    if len(prefs.ProjectNames) > 0 {
        // "Known projects for this user: Infinite Brain, FocaApp, ..."
    }
}
```

## Minimum Data Threshold

The preference model is only activated after:
- 50+ AI classifications by the user
- 10+ corrections made

Below this threshold, base model behavior. Above: personalized prompts.

## Privacy

The preference profile is:
- User-owned: visible via `GET /api/v1/preferences/ai`
- Deletable: `DELETE /api/v1/preferences/ai` clears all signals
- Never shared between users or orgs
- Encrypted at rest (T-104)

## Acceptance Criteria

- [ ] Correction events captured for all 5 signal types
- [ ] Preference profile built from corrections after threshold met
- [ ] Profile injected into classify/tag prompts when above threshold
- [ ] `GET /api/v1/preferences/ai` returns current profile
- [ ] `DELETE /api/v1/preferences/ai` clears all corrections and profile
- [ ] Profile encrypted at rest
- [ ] Integration test: 50 corrections → profile built → next classification uses profile
- [ ] 90% test coverage

## Dependencies

- T-120 (Event sourcing — correction events)
- T-122 (Prompt versioning — preferences injected into prompts)
- T-104 (Compliance — profile encrypted at rest)
