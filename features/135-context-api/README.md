# T-135 — Context API (External AI Tools Query IB)

## Overview

IB exposes a query endpoint that external AI coding tools (Cursor, Claude Code, GitHub Copilot,
custom agents) can call to retrieve domain context before generating code.

Instead of the AI tool hallucinating business rules or architectural decisions, it queries IB
first. IB returns the relevant business rules, ADRs, requirements, and past decisions.
The AI tool then generates code with full domain awareness.

This is "Context as a Service."

## The Problem

AI coding tools have access to code but not to the WHY behind it:
- Why is this endpoint rate-limited differently?
- What are the authentication rules for this service?
- Does this new feature conflict with any existing business rules?
- What did we decide about how to handle PHI in this domain?

Without this context, AI tools make wrong assumptions. With IB, they can ask.

## API

```
GET /api/v1/context?query=<natural language query>&categories=<comma-separated>&limit=10
```

### Request

```
GET /api/v1/context?query=how+should+JWT+authentication+work&categories=auth,security&limit=5
Authorization: Bearer <token>   (or API key for tool integrations)
```

### Response

```json
{
  "query": "how should JWT authentication work",
  "results": [
    {
      "type": "business_rule",
      "id": "...",
      "title": "JWT tokens must expire in 30 days",
      "body": "All JWTs issued by the system must have an exp claim set to 30 days from issuance. Refresh tokens extend this window.",
      "severity": "must",
      "source": "ADR-007",
      "relevance": 0.97
    },
    {
      "type": "adr",
      "id": "...",
      "title": "ADR-007: JWT over sessions",
      "body": "We chose JWT over server-side sessions because the system is stateless by design...",
      "relevance": 0.94
    },
    {
      "type": "requirement",
      "id": "...",
      "title": "Authentication must support refresh token rotation",
      "body": "...",
      "acceptance_criteria": ["Given a valid refresh token, when rotated, then the old token is invalidated immediately"],
      "relevance": 0.88
    }
  ],
  "token_count": 847
}
```

## Claude Code Integration (MCP)

IB can be registered as an MCP server for Claude Code. This allows Claude Code to query IB
automatically before generating code in a project:

```json
// .mcp.json
{
  "mcpServers": {
    "infinite-brain": {
      "type": "http",
      "url": "https://infinitebrain.io/mcp",
      "token": "${IB_API_KEY}"
    }
  }
}
```

With this configured, Claude Code automatically has access to:
- `get_business_rules(category)` — retrieve rules for a domain
- `query_context(query)` — semantic search over all knowledge
- `get_requirements(task)` — get acceptance criteria for a task
- `check_conflicts(description)` — check if a new rule/design conflicts with existing ones

## Cursor / Copilot Integration

Via `.cursorrules` or a custom instruction file that calls the IB context endpoint:

```
Before generating code for authentication-related tasks, retrieve context from:
https://infinitebrain.io/api/v1/context?query={task_description}&categories=auth,security
Include the returned rules and requirements in your response.
```

## Rate Limiting

Context API is metered separately from personal AI calls:
- Free tier: 100 context queries/month
- Pro: 1,000/month
- Teams: unlimited (included in team subscription)
- External apps: API key with custom rate limits

## Acceptance Criteria

- [ ] `GET /api/v1/context` endpoint with natural language query
- [ ] Returns ranked results (rules, ADRs, requirements) by relevance
- [ ] Token count included in response (helps tool providers stay within context limits)
- [ ] API key auth (separate from user JWT — for tool integrations)
- [ ] MCP server implementation (IB is queryable as an MCP resource)
- [ ] Rate limiting per API key
- [ ] Response cached for identical queries (5 min TTL)
- [ ] 90% test coverage

## Dependencies

- T-023 (semantic search — relevance ranking)
- T-131 (business rules — primary content type)
- T-132 (requirements — returned in results)
- T-020 (AI provider — re-ranking results)
- T-098 (security — API key management)
