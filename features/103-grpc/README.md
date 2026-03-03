# T-103 — gRPC + Protocol Buffers

## Overview

Add gRPC as the internal service communication layer and foundation for the external SDK. REST (Huma v2) remains the external user-facing API. gRPC handles two distinct use cases:

1. **Internal workers**: AI workers, notification dispatch, background job coordination
2. **External SDK**: Mobile clients (iOS/Android), CLI tools, and enterprise integrations that need a typed, generated SDK rather than REST

Proto files are the single source of truth for service contracts. Code generation is automated via `buf`.

---

## Why gRPC Alongside REST

| Layer | Protocol | Why |
|---|---|---|
| External user API | REST (Huma v2) | Browser-friendly, OpenAPI generated, simple to test with curl |
| Internal service communication | gRPC | Streaming, bi-directional, type-safe, low overhead |
| External SDK clients | gRPC | Generated clients in Go, Swift, Kotlin — no hand-written SDK |
| MCP server ↔ AI workers | gRPC | Streaming completions, cancellation, backpressure |

This is not a microservices split. The server runs as a single binary with both HTTP and gRPC listeners on different ports.

---

## Server Setup

```
Ports:
  :8090  — REST API (Huma v2)
  :8091  — MCP server (stdio or HTTP/SSE)
  :9090  — gRPC (internal + SDK)
```

```go
// cmd/server/main.go

func main() {
    // HTTP server (Huma v2) on :8090
    httpServer := setupHTTP(deps)

    // gRPC server on :9090
    grpcServer := setupGRPC(deps)

    // MCP server on :8091 (or stdio)
    mcpServer := setupMCP(deps)

    // Run all three concurrently with graceful shutdown
    runServers(ctx, httpServer, grpcServer, mcpServer)
}
```

---

## Proto File Structure

```
api/
└── proto/
    ├── buf.yaml                  # buf configuration
    ├── buf.gen.yaml              # code generation config
    ├── common/
    │   └── v1/
    │       └── types.proto       # shared: UUID, Pagination, Error
    ├── capture/
    │   └── v1/
    │       └── capture.proto     # CaptureService
    ├── ai/
    │   └── v1/
    │       └── ai.proto          # AIService (streaming completions)
    ├── knowledge/
    │   └── v1/
    │       └── knowledge.proto   # KnowledgeService (graph queries)
    ├── memory/
    │   └── v1/
    │       └── memory.proto      # MemoryService (agent memories)
    └── notification/
        └── v1/
            └── notification.proto  # NotificationService
```

Generated output goes to `internal/gen/` (do not edit manually).

---

## Common Types

```protobuf
// api/proto/common/v1/types.proto
syntax = "proto3";
package common.v1;
option go_package = "infinitebrain.io/internal/gen/common/v1;commonv1";

message UUID {
  string value = 1; // canonical UUID string format
}

message Pagination {
  string cursor = 1;
  int32  limit  = 2;
}

message Error {
  string code    = 1;
  string message = 2;
}
```

---

## Capture Service

```protobuf
// api/proto/capture/v1/capture.proto
syntax = "proto3";
package capture.v1;
option go_package = "infinitebrain.io/internal/gen/capture/v1;capturev1";

import "common/v1/types.proto";
import "google/protobuf/timestamp.proto";

service CaptureService {
  // Called by AI workers after transcription completes
  rpc ProcessCapture(ProcessCaptureRequest) returns (ProcessCaptureResponse);

  // Stream processing status back to caller (for long-running AI pipeline)
  rpc StreamProcessing(StreamProcessingRequest) returns (stream ProcessingEvent);
}

message ProcessCaptureRequest {
  common.v1.UUID user_id    = 1;
  common.v1.UUID capture_id = 2;
  string         content    = 3;  // text content (already transcribed if audio)
  string         source     = 4;  // "api" | "telegram" | "email" | "webhook"
}

message ProcessCaptureResponse {
  common.v1.UUID node_id  = 1;
  string         para     = 2;  // classified PARA category
  repeated string tags    = 3;
}

message StreamProcessingRequest {
  common.v1.UUID capture_id = 1;
}

message ProcessingEvent {
  string stage   = 1;  // "transcribe" | "classify" | "tag" | "embed" | "link" | "done"
  string status  = 2;  // "started" | "completed" | "failed"
  string detail  = 3;
  google.protobuf.Timestamp timestamp = 4;
}
```

---

## AI Service

```protobuf
// api/proto/ai/v1/ai.proto
syntax = "proto3";
package ai.v1;
option go_package = "infinitebrain.io/internal/gen/ai/v1;aiv1";

service AIService {
  // Unary completion (classify, tag, summarize)
  rpc Complete(CompleteRequest) returns (CompleteResponse);

  // Streaming completion (Q&A, digest generation — token-by-token)
  rpc StreamComplete(CompleteRequest) returns (stream CompleteChunk);

  // Generate embedding vector
  rpc Embed(EmbedRequest) returns (EmbedResponse);
}

message Message {
  string role    = 1;  // "user" | "assistant" | "system"
  string content = 2;
}

message CompleteRequest {
  repeated Message messages = 1;
  string           system   = 2;
  int32            max_tokens   = 3;
  float            temperature  = 4;
}

message CompleteResponse {
  string content         = 1;
  int32  input_tokens    = 2;
  int32  output_tokens   = 3;
}

message CompleteChunk {
  string delta     = 1;
  bool   is_final  = 2;
}

message EmbedRequest {
  string text = 1;
}

message EmbedResponse {
  repeated float embedding = 1;
}
```

---

## Knowledge Service (External SDK)

This service is exposed to SDK clients (mobile apps, CLI). It provides typed, versioned access to the knowledge graph.

```protobuf
// api/proto/knowledge/v1/knowledge.proto
syntax = "proto3";
package knowledge.v1;
option go_package = "infinitebrain.io/internal/gen/knowledge/v1;knowledgev1";

import "common/v1/types.proto";
import "google/protobuf/timestamp.proto";

service KnowledgeService {
  rpc CreateNode(CreateNodeRequest) returns (CreateNodeResponse);
  rpc GetNode(GetNodeRequest)       returns (Node);
  rpc ListNodes(ListNodesRequest)   returns (ListNodesResponse);
  rpc SearchNodes(SearchRequest)    returns (SearchResponse);
  rpc DeleteNode(DeleteNodeRequest) returns (DeleteNodeResponse);
}

message Node {
  common.v1.UUID node_id    = 1;
  string         type       = 2;
  string         title      = 3;
  string         content    = 4;
  string         para       = 5;
  repeated string tags      = 6;
  common.v1.UUID project_id = 7;
  google.protobuf.Timestamp created_at = 8;
}

message CreateNodeRequest {
  string type    = 1;
  string title   = 2;
  string content = 3;
  string para    = 4;
  common.v1.UUID project_id = 5;
}

message CreateNodeResponse {
  common.v1.UUID node_id = 1;
}

message GetNodeRequest {
  common.v1.UUID node_id = 1;
}

message ListNodesRequest {
  string             type       = 1;  // optional filter
  common.v1.UUID     project_id = 2;  // optional filter
  common.v1.Pagination pagination = 3;
}

message ListNodesResponse {
  repeated Node nodes       = 1;
  string        next_cursor = 2;
}

message SearchRequest {
  string             query      = 1;
  common.v1.UUID     project_id = 2;  // optional: scope to project
  int32              limit      = 3;
}

message SearchResponse {
  repeated SearchResult results = 1;
}

message SearchResult {
  Node  node       = 1;
  float similarity = 2;
}

message DeleteNodeRequest {
  common.v1.UUID node_id = 1;
}

message DeleteNodeResponse {
  bool deleted = 1;
}
```

---

## Memory Service

Used by AI workers to read/write agent memories.

```protobuf
// api/proto/memory/v1/memory.proto
syntax = "proto3";
package memory.v1;
option go_package = "infinitebrain.io/internal/gen/memory/v1;memoryv1";

import "common/v1/types.proto";
import "google/protobuf/timestamp.proto";

service MemoryService {
  rpc StoreMemory(StoreMemoryRequest)   returns (StoreMemoryResponse);
  rpc LoadContext(LoadContextRequest)   returns (LoadContextResponse);
  rpc SearchMemories(SearchMemoriesRequest) returns (SearchMemoriesResponse);
}

message StoreMemoryRequest {
  common.v1.UUID session_id  = 1;
  string         agent_id    = 2;
  common.v1.UUID project_id  = 3;
  string         type        = 4;  // observation | decision | context | pattern | error
  string         content     = 5;
  float          confidence  = 6;
  google.protobuf.Timestamp expires_at = 7;
}

message StoreMemoryResponse {
  common.v1.UUID memory_id = 1;
}

message LoadContextRequest {
  common.v1.UUID user_id    = 1;
  common.v1.UUID project_id = 2;
  string         query      = 3;  // for semantic similarity ranking
  int32          limit      = 4;
}

message LoadContextResponse {
  repeated Memory memories = 1;
}

message Memory {
  common.v1.UUID memory_id  = 1;
  string         agent_id   = 2;
  string         type       = 3;
  string         content    = 4;
  float          confidence = 5;
  google.protobuf.Timestamp created_at = 6;
}

message SearchMemoriesRequest {
  common.v1.UUID user_id    = 1;
  common.v1.UUID project_id = 2;
  string         query      = 3;
  int32          limit      = 4;
}

message SearchMemoriesResponse {
  repeated Memory memories = 1;
}
```

---

## buf Configuration

```yaml
# api/proto/buf.yaml
version: v2
modules:
  - path: .
    name: buf.build/infinitebrain/api
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

```yaml
# api/proto/buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: ../../internal/gen
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go
    out: ../../internal/gen
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false
```

---

## gRPC Server Setup

```go
// internal/grpc/server.go

type Server struct {
    captureService    capture.CaptureServiceServer
    aiService         ai.AIServiceServer
    knowledgeService  knowledge.KnowledgeServiceServer
    memoryService     memory.MemoryServiceServer
    auth              auth.Authenticator
    logger            *slog.Logger
}

func NewServer(
    captureService    capturev1.CaptureServiceServer,
    aiService         aiv1.AIServiceServer,
    knowledgeService  knowledgev1.KnowledgeServiceServer,
    memoryService     memoryv1.MemoryServiceServer,
    auth              auth.Authenticator,
    logger            *slog.Logger,
) *grpc.Server {
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            otelgrpc.UnaryServerInterceptor(),
            grpcAuth(auth),
            grpcLogger(logger),
        ),
        grpc.ChainStreamInterceptor(
            otelgrpc.StreamServerInterceptor(),
            grpcStreamAuth(auth),
        ),
    )
    capturev1.RegisterCaptureServiceServer(s, captureService)
    aiv1.RegisterAIServiceServer(s, aiService)
    knowledgev1.RegisterKnowledgeServiceServer(s, knowledgeService)
    memoryv1.RegisterMemoryServiceServer(s, memoryService)

    // Enable reflection for grpcurl and dev tooling
    reflection.Register(s)

    return s
}
```

---

## Auth Interceptor

gRPC auth mirrors the REST auth — same `Authenticator` interface (T-100).

```go
// internal/grpc/auth.go

func grpcAuth(authenticator auth.Authenticator) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization")
        }
        token := strings.TrimPrefix(tokens[0], "Bearer ")
        claims, err := authenticator.Validate(ctx, token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        ctx = auth.WithClaims(ctx, claims)
        return handler(ctx, req)
    }
}
```

---

## Makefile Targets

```makefile
# Code generation
.PHONY: proto
proto:
	cd api/proto && buf generate

# Lint proto files
.PHONY: proto-lint
proto-lint:
	cd api/proto && buf lint

# Check for breaking changes against last commit
.PHONY: proto-breaking
proto-breaking:
	cd api/proto && buf breaking --against '.git#branch=main'

# Install buf
.PHONY: install-buf
install-buf:
	go install github.com/bufbuild/buf/cmd/buf@latest
```

---

## Dependencies

```
google.golang.org/grpc          v1.64+
google.golang.org/protobuf      v1.34+
github.com/bufbuild/buf         (CLI tool, not a Go dependency)
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
```

---

## Breaking Change Policy

Proto changes must be backward-compatible. `buf breaking` runs in CI against `main`:

- Adding fields: always safe (proto3 default values)
- Removing fields: **breaking** — mark deprecated, remove in next major version
- Renaming services/RPCs: **breaking** — never rename
- Changing field types: **breaking** — never do this

---

## Acceptance Criteria

- [ ] `api/proto/` directory with `buf.yaml` and `buf.gen.yaml`
- [ ] Proto files for: `common/v1`, `capture/v1`, `ai/v1`, `knowledge/v1`, `memory/v1`
- [ ] `make proto` generates Go code into `internal/gen/` (no manual edits to gen/)
- [ ] `make proto-lint` passes with no violations
- [ ] `make proto-breaking` runs in CI (checked against main branch)
- [ ] gRPC server listens on `:9090`, registered in `cmd/server/main.go`
- [ ] Auth interceptor reuses `Authenticator` interface (T-100/T-007)
- [ ] OpenTelemetry tracing interceptor on all gRPC calls
- [ ] gRPC reflection enabled (dev tooling via `grpcurl`)
- [ ] `KnowledgeService` — all 5 RPCs implemented and tested
- [ ] `AIService.StreamComplete` streams tokens back correctly
- [ ] `MemoryService` — StoreMemory, LoadContext, SearchMemories implemented
- [ ] Unit tests for each gRPC service handler
- [ ] Integration test: start gRPC server, call RPCs with real DB
- [ ] 90% test coverage

---

## Dependencies

- T-007 (Auth — `Authenticator` interface, auth interceptor reuses it)
- T-028 (Knowledge graph — `KnowledgeService` queries `nodes` + `edges`)
- T-016 (AI session memory — `MemoryService` backed by `agent_memories`)
- T-020 (AI provider — `AIService` delegates to `Provider` interface)

## Notes

- `internal/gen/` is committed to the repo (generated code, read-only). This avoids requiring `buf` at build time in CI — only needed when proto files change.
- gRPC-Web (for browser clients) is a `someday` task — browsers can't speak raw gRPC. For now, browsers use REST.
- The `KnowledgeService` is the first gRPC service exposed to external SDK consumers. Its stability matters most — follow the breaking change policy strictly.
- Server port `:9090` — no conflict with REST (`:8090`) or MCP (`:8091`)
