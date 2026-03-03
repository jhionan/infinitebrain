# Feature: AI Auto-Classification

**Task ID**: T-021
**Status**: planned
**Epic**: AI Engine

## Goal

Automatically classify captured notes into the PARA categories (Projects, Areas,
Resources, Archives) and link them to the correct project or area using AI.

## Acceptance Criteria

- [ ] `internal/ai/classify.go` — Classification service
- [ ] `internal/ai/prompts/classify.go` — Versioned prompts
- [ ] Classification triggered via Asynq job after note creation
- [ ] Structured JSON output from AI (validated before saving)
- [ ] Confidence score stored alongside classification
- [ ] Low-confidence items flagged for user review instead of auto-classified
- [ ] User can override classification (feedback loop)
- [ ] Unit tests with mocked AI provider
- [ ] Classification accuracy metric logged

## Classification Output

```go
type ClassificationResult struct {
    PARACategory  PARACategory    // project | area | resource | archive
    ProjectID     *uuid.UUID      // matched project, if any
    AreaID        *uuid.UUID      // matched area, if any
    Tags          []string        // extracted tags
    Title         string          // AI-generated title (if original was empty)
    Summary       string          // 1-2 sentence summary
    Confidence    float32         // 0.0 - 1.0
    ShouldReview  bool            // true if confidence < 0.7
    Entities      []Entity        // people, places, tools mentioned
}

type Entity struct {
    Type  string // person | organization | tool | date | location
    Value string
}
```

## Classification Prompt Strategy

The prompt includes:
1. User's existing projects and areas (for context matching)
2. Recent notes (for personal vocabulary learning)
3. The note content
4. Explicit JSON output schema

Confidence threshold:
- `≥ 0.8` → auto-classify and move from inbox
- `0.6 - 0.79` → classify but flag for review
- `< 0.6` → leave in inbox with suggested classification

## Processing Pipeline

```
Note created
    ↓
Asynq job: ai:process_capture
    ↓
1. Fetch user's projects + areas
2. Build classification prompt
3. Call AI provider
4. Parse + validate JSON response
5. Update note with classification
6. Generate and store embedding
7. Notify user if review required
```
