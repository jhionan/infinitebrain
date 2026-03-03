# T-097 — MCP Provider Adapter

## Overview

Implement the `Provider` interface (T-020) backed by an external MCP server. This allows any MCP-compatible AI model — a self-hosted Ollama instance, a custom fine-tuned model, a remote GPT-4 wrapper — to power Infinite Brain's AI pipeline (classification, tagging, embedding, Q&A, digests) instead of Claude or OpenAI.

Users bring their own AI brain and plug it in via a single config line.

---

## Why

Infinite Brain's T-020 `Provider` interface abstracts the AI layer. Concrete implementations today are `ClaudeProvider` and `OpenAIProvider`. Adding `MCPProvider` means any AI that speaks MCP can be the brain behind the brain.

This is the "pluggable AI brain" capability. A user running a local Llama 3 model via Ollama shouldn't be forced to use the Anthropic API. A team with a fine-tuned model on business rules should be able to wire it directly into Infinite Brain's classification pipeline.

---

## Provider Interface (from T-020)

```go
// internal/ai/provider.go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Transcribe(ctx context.Context, audio io.Reader) (string, error)
}
```

`MCPProvider` implements all three methods by calling tools on an external MCP server.

---

## What the External MCP Server Must Expose

For an external AI to serve as a Provider, its MCP server must expose at minimum:

| Tool | Maps to | Required |
|---|---|---|
| `complete` | `Provider.Complete` | yes |
| `embed` | `Provider.Embed` | yes |
| `transcribe` | `Provider.Transcribe` | no — falls back to OpenAI Whisper |

Tool schemas:

```json
// complete
{
  "name": "complete",
  "inputSchema": {
    "type": "object",
    "required": ["messages"],
    "properties": {
      "messages":    { "type": "array", "items": { "type": "object" } },
      "system":      { "type": "string" },
      "max_tokens":  { "type": "integer" },
      "temperature": { "type": "number" }
    }
  }
}

// embed
{
  "name": "embed",
  "inputSchema": {
    "type": "object",
    "required": ["text"],
    "properties": {
      "text": { "type": "string" }
    }
  }
}

// transcribe (optional)
{
  "name": "transcribe",
  "inputSchema": {
    "type": "object",
    "required": ["audio_base64"],
    "properties": {
      "audio_base64": { "type": "string" },
      "language":     { "type": "string" }
    }
  }
}
```

This is a minimal contract. The external server can be Ollama with an MCP wrapper, a custom Python server, or any MCP-compatible AI endpoint.

---

## Implementation

```go
// internal/ai/providers/mcp_provider.go

type MCPProvider struct {
    client     MCPClient    // mcp-go client
    serverURL  string
    transport  string       // "stdio" | "http"
    command    []string     // for stdio: e.g. ["ollama", "mcp"]
    logger     zerolog.Logger
}

func NewMCPProvider(cfg MCPProviderConfig, logger zerolog.Logger) (*MCPProvider, error) {
    var client MCPClient
    switch cfg.Transport {
    case "stdio":
        client = mcp.NewStdioClient(cfg.Command[0], cfg.Command[1:]...)
    case "http":
        client = mcp.NewHTTPClient(cfg.ServerURL, cfg.APIKey)
    }
    if err := client.Initialize(context.Background()); err != nil {
        return nil, fmt.Errorf("mcp provider init: %w", err)
    }
    return &MCPProvider{client: client, logger: logger}, nil
}

func (p *MCPProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    result, err := p.client.CallTool(ctx, "complete", map[string]any{
        "messages":    req.Messages,
        "system":      req.System,
        "max_tokens":  req.MaxTokens,
        "temperature": req.Temperature,
    })
    if err != nil {
        return CompletionResponse{}, fmt.Errorf("mcp complete: %w", err)
    }
    return parseCompletionResult(result), nil
}

func (p *MCPProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    result, err := p.client.CallTool(ctx, "embed", map[string]any{"text": text})
    if err != nil {
        return nil, fmt.Errorf("mcp embed: %w", err)
    }
    return parseEmbedResult(result), nil
}

func (p *MCPProvider) Transcribe(ctx context.Context, audio io.Reader) (string, error) {
    // Check if remote server supports transcription
    if !p.supportsTranscribe {
        return "", ErrNotSupported
    }
    b, err := io.ReadAll(audio)
    if err != nil {
        return "", fmt.Errorf("reading audio: %w", err)
    }
    result, err := p.client.CallTool(ctx, "transcribe", map[string]any{
        "audio_base64": base64.StdEncoding.EncodeToString(b),
    })
    if err != nil {
        return "", fmt.Errorf("mcp transcribe: %w", err)
    }
    return parseTranscribeResult(result), nil
}
```

---

## Capability Discovery

On initialization, `MCPProvider` calls `ListTools` on the external server to discover what it supports:

```go
func (p *MCPProvider) discoverCapabilities(ctx context.Context) error {
    tools, err := p.client.ListTools(ctx)
    if err != nil {
        return fmt.Errorf("listing tools: %w", err)
    }
    toolNames := make(map[string]bool)
    for _, t := range tools {
        toolNames[t.Name] = true
    }
    p.supportsComplete    = toolNames["complete"]
    p.supportsEmbed       = toolNames["embed"]
    p.supportsTranscribe  = toolNames["transcribe"]

    if !p.supportsComplete {
        return fmt.Errorf("external MCP server must support the 'complete' tool")
    }
    return nil
}
```

If `embed` is not supported, the system falls back to the configured fallback provider (e.g., OpenAI text-embedding-3-small). If `transcribe` is not supported, falls back to OpenAI Whisper. Only `complete` is required.

---

## Fallback Chain

```go
// internal/ai/provider_chain.go

type ProviderChain struct {
    primary  Provider
    fallback Provider
}

func (c *ProviderChain) Embed(ctx context.Context, text string) ([]float32, error) {
    result, err := c.primary.Embed(ctx, text)
    if errors.Is(err, ErrNotSupported) {
        return c.fallback.Embed(ctx, text)
    }
    return result, err
}
```

The chain is constructed at startup based on config. If the primary is `MCPProvider` and it doesn't support embeddings, the chain falls through to OpenAI automatically.

---

## Configuration

```yaml
# configs/example.env

# Use a local Ollama model via stdio MCP wrapper
AI_PROVIDER=mcp
AI_MCP_TRANSPORT=stdio
AI_MCP_COMMAND=ollama-mcp   # binary that wraps Ollama as an MCP server

# Or a remote MCP server over HTTP
AI_PROVIDER=mcp
AI_MCP_TRANSPORT=http
AI_MCP_SERVER_URL=http://my-ai-server:8090
AI_MCP_API_KEY=sk-...

# Fallback for capabilities the primary doesn't support
AI_EMBED_FALLBACK=openai
AI_TRANSCRIBE_FALLBACK=openai
```

---

## Provider Registration

```go
// internal/ai/factory.go

func NewProvider(cfg config.Config, logger zerolog.Logger) (Provider, error) {
    switch cfg.AI.Provider {
    case "claude":
        return NewClaudeProvider(cfg, logger)
    case "openai":
        return NewOpenAIProvider(cfg, logger)
    case "mcp":
        primary, err := NewMCPProvider(cfg.AI.MCP, logger)
        if err != nil {
            return nil, err
        }
        fallback, _ := NewOpenAIProvider(cfg, logger)
        return NewProviderChain(primary, fallback), nil
    default:
        return nil, fmt.Errorf("unknown AI provider: %s", cfg.AI.Provider)
    }
}
```

---

## Example: Ollama Integration

A user running Llama 3 locally:

```bash
# Install an Ollama MCP wrapper (community tool or custom)
go install github.com/example/ollama-mcp@latest

# Configure Infinite Brain
export AI_PROVIDER=mcp
export AI_MCP_TRANSPORT=stdio
export AI_MCP_COMMAND=ollama-mcp

# Start the server — AI pipeline now runs through local Llama 3
make run
```

Infinite Brain's classify, tag, Q&A, and digest all run through Ollama. Zero data leaves the machine.

---

## Acceptance Criteria

- [ ] `MCPProvider` implements `Provider` interface (Complete, Embed, Transcribe)
- [ ] stdio transport: connects to external process via stdin/stdout
- [ ] HTTP transport: connects to remote MCP server with optional API key
- [ ] `discoverCapabilities` called on init; logs which capabilities are available
- [ ] `complete` not supported → initialization fails with clear error
- [ ] `embed` not supported → falls back to configured fallback provider
- [ ] `transcribe` not supported → falls back to configured fallback provider
- [ ] `ProviderChain` routes to fallback on `ErrNotSupported`
- [ ] `NewProvider` factory returns `MCPProvider` when `AI_PROVIDER=mcp`
- [ ] Unit tests for MCPProvider with a mock MCP server
- [ ] Unit tests for ProviderChain fallback behavior
- [ ] Integration test: start a minimal MCP server stub, call Complete via MCPProvider
- [ ] 90% test coverage

---

## Dependencies

- T-020 (AI provider abstraction — the interface being implemented)

## Notes

- `MCPProvider` does not call Infinite Brain's own MCP server (T-096) — it calls an external third-party MCP server that provides AI capabilities. These are inverse directions.
- The MCP client library is the same `mark3labs/mcp-go` used in T-096, but used as a client, not a server.
- Embedding dimension from external providers may differ from OpenAI's 1536. The `nodes.embedding` column should be created with the configured dimension. Document this constraint clearly — changing dimension after data exists requires re-embedding all nodes.
- Timeout config: external MCP servers may be slow (local model inference). Default timeout: 120s for Complete, 30s for Embed.
