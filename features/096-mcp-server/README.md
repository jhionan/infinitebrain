# T-096 — MCP Server

## Overview

Expose Infinite Brain as a Model Context Protocol (MCP) server. Any MCP-compatible AI client — Claude Code, Cursor, custom agents — can connect and use your brain as a live tool: capturing notes, searching your knowledge graph, loading memory context, managing tasks, and controlling the daily chunk planner.

The MCP server is a separate binary (`cmd/mcp/main.go`) that calls the same service layer as the REST API. It is another interface on top of the same system.

---

## Why

MCP turns Infinite Brain from a personal app into an **AI-native interface**. Instead of an AI asking you "what context do you have on this?", it reaches directly into your brain. Instead of manually copying a task into a chat, the AI creates it. The second brain becomes the AI's first resource.

Two transports serve different use cases:
- **stdio** — for Claude Code and local agents (zero config, no auth needed)
- **HTTP/SSE** — for remote agents, custom scripts, any non-local client

---

## Entry Point

```
cmd/mcp/main.go
```

Separate from `cmd/server/main.go`. Shares all service dependencies. Can run alongside the REST API or standalone.

```go
// cmd/mcp/main.go
func main() {
    cfg := config.Load()
    logger := logger.New(cfg)
    db := database.Connect(cfg)

    // Wire services (same as REST server)
    nodeService    := knowledge.NewNodeService(...)
    memoryService  := ai.NewMemoryService(...)
    taskService    := capture.NewTaskService(...)
    plannerService := adhd.NewPlannerService(...)

    srv := mcp.NewServer(cfg, logger, nodeService, memoryService, taskService, plannerService)

    switch cfg.MCP.Transport {
    case "stdio":
        srv.ServeStdio()
    case "http":
        srv.ServeHTTP(cfg.MCP.Addr)
    }
}
```

---

## Library

```
github.com/mark3labs/mcp-go
```

Most mature Go MCP library. Supports stdio and HTTP/SSE transports, tool registration, resource registration, and schema validation.

```go
go get github.com/mark3labs/mcp-go
```

---

## Internal Structure

```
internal/mcp/
├── server.go           — server init, tool + resource registration
├── auth.go             — API key middleware (HTTP transport only)
├── tools/
│   ├── capture.go      — infinitebrain__capture, infinitebrain__capture_voice
│   ├── knowledge.go    — search, get_node, link_nodes, get_insights
│   ├── memory.go       — store_memory, load_context
│   ├── tasks.go        — get_tasks, create_task
│   ├── planner.go      — get_today, start_chunk, complete_chunk, suggest_task
│   └── review.go       — get_review_queue, review_node
└── resources/
    ├── nodes.go        — infinitebrain://nodes/{id}
    ├── today.go        — infinitebrain://today
    ├── projects.go     — infinitebrain://projects
    └── insights.go     — infinitebrain://insights/recent
```

---

## Tools

All tools are prefixed `infinitebrain__`. Input/output schemas use JSON Schema.

### Capture

#### `infinitebrain__capture`
Create any node in the knowledge graph.

```json
{
  "name": "infinitebrain__capture",
  "description": "Capture anything into Infinite Brain — a note, task, idea, movie, book, event, business rule, decision, or any other entity.",
  "inputSchema": {
    "type": "object",
    "required": ["title", "type"],
    "properties": {
      "title":      { "type": "string", "description": "Short title or headline" },
      "type":       { "type": "string", "enum": ["note","task","event","media","rule","decision","contact","place"] },
      "content":    { "type": "string", "description": "Full content or description" },
      "para":       { "type": "string", "enum": ["project","area","resource","archive"] },
      "project_id": { "type": "string", "description": "UUID of the project node this belongs to" },
      "metadata":   { "type": "object", "description": "Type-specific metadata (e.g. scheduled_at for events, genre for media)" }
    }
  }
}
```

Response: `{ "node_id": "uuid", "title": "...", "review_stage": 0, "next_review_at": "..." }`

---

#### `infinitebrain__capture_voice`
Transcribe audio and capture as a note.

```json
{
  "name": "infinitebrain__capture_voice",
  "description": "Transcribe audio content and save as a note in Infinite Brain.",
  "inputSchema": {
    "type": "object",
    "required": ["audio_base64"],
    "properties": {
      "audio_base64": { "type": "string", "description": "Base64-encoded audio file (mp3, m4a, wav)" },
      "project_id":   { "type": "string" }
    }
  }
}
```

---

### Knowledge

#### `infinitebrain__search`
Semantic search across all nodes.

```json
{
  "name": "infinitebrain__search",
  "description": "Search your knowledge graph semantically. Returns the most relevant nodes for a query.",
  "inputSchema": {
    "type": "object",
    "required": ["query"],
    "properties": {
      "query":      { "type": "string" },
      "type":       { "type": "string", "description": "Filter by node type" },
      "project_id": { "type": "string", "description": "Limit to a specific project" },
      "limit":      { "type": "integer", "default": 10 }
    }
  }
}
```

Response: array of `{ node_id, title, type, content_snippet, score, project_id }`

---

#### `infinitebrain__get_node`
Retrieve a node with its edges.

```json
{
  "name": "infinitebrain__get_node",
  "description": "Get a specific node from the knowledge graph including its relationships to other nodes.",
  "inputSchema": {
    "type": "object",
    "required": ["node_id"],
    "properties": {
      "node_id":    { "type": "string" },
      "graph_depth": { "type": "integer", "default": 1, "description": "How many hops of edges to include" }
    }
  }
}
```

---

#### `infinitebrain__link_nodes`
Create a typed edge between two nodes.

```json
{
  "name": "infinitebrain__link_nodes",
  "description": "Create a relationship between two nodes in the knowledge graph.",
  "inputSchema": {
    "type": "object",
    "required": ["from_node_id", "to_node_id", "relation_type"],
    "properties": {
      "from_node_id":  { "type": "string" },
      "to_node_id":    { "type": "string" },
      "relation_type": { "type": "string", "enum": ["implements","solves","contradicts","relates","inspired_by","blocks","part_of"] },
      "confidence":    { "type": "number", "default": 1.0 }
    }
  }
}
```

---

#### `infinitebrain__get_insights`
Retrieve recent cross-project insights.

```json
{
  "name": "infinitebrain__get_insights",
  "description": "Get recent insights discovered by the cross-project insight linker — unexpected connections between ideas across different projects.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "limit": { "type": "integer", "default": 10 }
    }
  }
}
```

---

### Memory

#### `infinitebrain__store_memory`
Persist AI reasoning for future sessions.

```json
{
  "name": "infinitebrain__store_memory",
  "description": "Store a piece of reasoning, observation, or context from this AI session so it persists for future sessions.",
  "inputSchema": {
    "type": "object",
    "required": ["content", "type"],
    "properties": {
      "content":    { "type": "string" },
      "type":       { "type": "string", "enum": ["observation","decision","context","pattern","error"] },
      "project_id": { "type": "string" },
      "confidence": { "type": "number", "default": 1.0 },
      "expires_at": { "type": "string", "description": "ISO 8601 datetime. Omit for permanent memory." }
    }
  }
}
```

---

#### `infinitebrain__load_context`
Load relevant memories and nodes for a query.

```json
{
  "name": "infinitebrain__load_context",
  "description": "Load relevant context from your knowledge graph and memory store for the current task or question. Call this at the start of a session to get up to speed.",
  "inputSchema": {
    "type": "object",
    "required": ["query"],
    "properties": {
      "query":      { "type": "string", "description": "What you're working on or asking about" },
      "project_id": { "type": "string" },
      "limit":      { "type": "integer", "default": 20 }
    }
  }
}
```

Response: `{ "memories": [...], "related_nodes": [...], "recent_decisions": [...] }`

This is the most important tool. An AI calls this at session start to load relevant context before doing any work.

---

### Tasks

#### `infinitebrain__get_tasks`

```json
{
  "name": "infinitebrain__get_tasks",
  "description": "List tasks ordered by priority. Optionally filter by project or status.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_id": { "type": "string" },
      "status":     { "type": "string", "enum": ["pending","in_progress","completed"] },
      "limit":      { "type": "integer", "default": 20 }
    }
  }
}
```

---

#### `infinitebrain__create_task`

```json
{
  "name": "infinitebrain__create_task",
  "description": "Create a task in Infinite Brain.",
  "inputSchema": {
    "type": "object",
    "required": ["title"],
    "properties": {
      "title":      { "type": "string" },
      "content":    { "type": "string" },
      "project_id": { "type": "string" },
      "priority":   { "type": "string", "enum": ["low","medium","high","urgent"] },
      "due_at":     { "type": "string", "description": "ISO 8601 datetime" }
    }
  }
}
```

---

### Daily Chunk Planner

#### `infinitebrain__get_today`

```json
{
  "name": "infinitebrain__get_today",
  "description": "Get today's chunk plan — how many chunks of each type remain, which is active, and overall progress.",
  "inputSchema": { "type": "object", "properties": {} }
}
```

Response:
```json
{
  "date": "2026-04-01",
  "total": 16,
  "completed": 7,
  "active_chunk": { "id": "...", "type": "work", "title": "Implement validateTransfer()", "started_at": "...", "duration_min": 60 },
  "remaining_by_type": { "work": 4, "chore": 2, "personal": 1, "free": 2 }
}
```

---

#### `infinitebrain__start_chunk`

```json
{
  "name": "infinitebrain__start_chunk",
  "description": "Start a pending chunk. For work chunks, optionally specify what task to work on.",
  "inputSchema": {
    "type": "object",
    "required": ["type"],
    "properties": {
      "type":    { "type": "string", "enum": ["work","chore","exercise","personal","free"] },
      "node_id": { "type": "string", "description": "Task node to work on (work chunks only)" },
      "title":   { "type": "string", "description": "What you'll work on, if no node_id" }
    }
  }
}
```

---

#### `infinitebrain__complete_chunk`

```json
{
  "name": "infinitebrain__complete_chunk",
  "description": "Mark the active chunk as complete.",
  "inputSchema": { "type": "object", "properties": {} }
}
```

---

#### `infinitebrain__suggest_task`

```json
{
  "name": "infinitebrain__suggest_task",
  "description": "Get AI-ranked task suggestions for the next work chunk, based on priority, current energy, and today's context.",
  "inputSchema": { "type": "object", "properties": {} }
}
```

---

### Review

#### `infinitebrain__get_review_queue`

```json
{
  "name": "infinitebrain__get_review_queue",
  "description": "Get items from the relevance review queue — things that haven't been confirmed relevant in a while.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "limit": { "type": "integer", "default": 10 }
    }
  }
}
```

---

#### `infinitebrain__review_node`

```json
{
  "name": "infinitebrain__review_node",
  "description": "Respond to a relevance review. 'yes' keeps the item (confirmed biannual), 'no' advances it toward deletion.",
  "inputSchema": {
    "type": "object",
    "required": ["node_id", "response"],
    "properties": {
      "node_id":  { "type": "string" },
      "response": { "type": "string", "enum": ["yes","no"] }
    }
  }
}
```

---

## Resources

Resources are read-only. AI clients use these to load context without calling a tool.

```
infinitebrain://nodes/{id}         Full node with edges
infinitebrain://today              Today's chunk plan
infinitebrain://projects           All projects (id, title, node count)
infinitebrain://insights/recent    Last 10 cross-project insights
```

---

## Authentication

### stdio transport
No auth. The process runs locally as the user. Suitable for Claude Code, local agents.

```json
// ~/.claude/mcp.json (Claude Code config)
{
  "mcpServers": {
    "infinitebrain": {
      "command": "infinite-brain-mcp",
      "args": ["--transport", "stdio"]
    }
  }
}
```

### HTTP/SSE transport
API key in `Authorization: Bearer <key>` header. Keys are user-scoped, managed via REST API:

```
POST /api/v1/mcp-keys        Generate an MCP API key
GET  /api/v1/mcp-keys        List active keys
DELETE /api/v1/mcp-keys/:id  Revoke a key
```

Keys stored hashed in a `mcp_api_keys` table. Validated in `internal/mcp/auth.go` middleware.

---

## Configuration

```yaml
# configs/example.env
MCP_TRANSPORT=stdio       # stdio | http
MCP_ADDR=:8081            # HTTP transport only
MCP_API_KEY_REQUIRED=true # HTTP transport only
```

---

## Acceptance Criteria

- [ ] `cmd/mcp/main.go` wires services and starts server (stdio or HTTP based on config)
- [ ] All 15 tools registered with correct JSON Schema input definitions
- [ ] All 4 resources registered and return correct data
- [ ] `infinitebrain__load_context` returns merged memories + related nodes
- [ ] stdio transport works with Claude Code (`claude mcp add infinitebrain -- infinite-brain-mcp`)
- [ ] HTTP transport validates API key before handling any request
- [ ] `POST /api/v1/mcp-keys` generates and stores a hashed API key
- [ ] Tool errors return MCP-compliant error responses (not panics)
- [ ] Unit tests for each tool handler
- [ ] Integration test: start stdio server, call `infinitebrain__capture`, verify node in DB
- [ ] 90% test coverage on `internal/mcp/`

---

## Dependencies

- T-004 (PostgreSQL)
- T-006 (HTTP server — shares service wiring pattern)
- T-007 (Auth — user context for all tool calls)
- T-016 (AI session memory — store_memory, load_context)
- T-028 (Knowledge graph — capture, search, get_node, link_nodes)
- T-030 (Tasks — get_tasks, create_task)
- T-048 (Chunk planner — get_today, start_chunk, complete_chunk, suggest_task)

## Notes

- Tool names use double underscore `infinitebrain__toolname` — MCP convention for namespaced tools
- The MCP server binary should be installable as `infinite-brain-mcp` for Claude Code integration
- All tools operate in the context of the authenticated user — no cross-user data access
- stdio transport is the primary target for MVP; HTTP/SSE is additive
