# Feature: AI Provider Abstraction

**Task ID**: T-020
**Status**: planned
**Epic**: AI Engine

## Goal

Create a provider-agnostic AI interface so all AI functionality works with Claude
(primary) or OpenAI (fallback) without changing business logic.

## Acceptance Criteria

- [ ] `internal/ai/provider.go` — Provider interface
- [ ] `internal/ai/anthropic.go` — Claude implementation
- [ ] `internal/ai/openai.go` — OpenAI implementation
- [ ] `internal/ai/factory.go` — Provider factory based on config
- [ ] Unit tests with mocked HTTP clients
- [ ] Error handling for rate limits, timeouts, and API errors

## Interface

```go
type Provider interface {
    // Complete sends a prompt and returns a text completion.
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

    // Embed generates a vector embedding for the given text.
    Embed(ctx context.Context, text string) ([]float32, error)

    // Transcribe converts audio to text (Whisper or equivalent).
    Transcribe(ctx context.Context, audio io.Reader, mimeType string) (string, error)
}

type CompletionRequest struct {
    SystemPrompt string
    UserPrompt   string
    MaxTokens    int
    Temperature  float32
    JSONMode     bool   // force JSON output
}

type CompletionResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
    Model        string
}
```

## Claude Implementation Notes

- Use `claude-sonnet-4-6` as default model
- Use streaming for long completions (daily digest, weekly review)
- JSON mode: use system prompt instructing JSON-only output + response validation
- Retry with exponential backoff on 429 (rate limit) and 529 (overloaded)

## Prompt Management

All prompts live in `internal/ai/prompts/` as typed constants:

```go
// internal/ai/prompts/classify.go
const ClassifyNoteV1 = `You are a personal knowledge manager...`
```

Prompts are versioned by appending `V1`, `V2`, etc. Old versions are never deleted
until the new version is validated in production.
