# T-028 — Knowledge Graph

## Overview

A graph layer that sits on top of all content in Infinite Brain. Every entity — note, task, movie, doctor appointment, business rule, book, code decision, insight — is a **node**. Connections between entities are **edges** with typed relationships.

This is what makes Infinite Brain more than a note-taking app. Notes accumulate. A knowledge graph compounds.

---

## Why

A software engineer working across multiple projects has business rules, decisions, and logic spread across their brain. A movie watchlist item and a code architecture decision live in the same system. The graph layer lets the AI reason about relationships: "this solution in Project A solves the problem you have in Project B", "the book you saved relates to the approach you used here".

The graph also enables the cross-project insight linker (T-029), semantic Q&A (T-024), and context restoration (T-046).

---

## Schema

> The canonical schema is defined in **T-004** (`features/004-database/README.md`).
> Reproduced here for reference; T-004 is the source of truth.

Key columns on `nodes` relevant to the knowledge graph:

| Column | Type | Purpose |
|---|---|---|
| `org_id` | UUID | Org isolation (RLS enforced) |
| `user_id` | UUID | Owning user |
| `type` | TEXT | Node type — see table below |
| `visibility` | TEXT | `personal \| team \| org \| public` — default `personal` |
| `is_phi` | BOOLEAN | Triggers encryption + audit logging |
| `embedding` | VECTOR(1536) | Semantic search via HNSW index |
| `search_vector` | TSVECTOR | Full-text search via GIN index |
| `para` | TEXT | PARA classification |
| `project_id` | UUID (self-ref) | Parent project node |
| `metadata` | JSONB | Node-type-specific fields (see below) |

Full DDL in `db/schema/002_nodes.hcl`.

### Node Types

| Type | PARA | Examples |
|---|---|---|
| `project` | project | "Payments API", "Home renovation" |
| `note` | any | Free-form captured thought |
| `task` | project/area | Actionable item with status |
| `event` | area | Doctor appointment, meeting |
| `media` | resource | Movie, book, podcast, article |
| `rule` | project | Business rule, constraint |
| `decision` | project | Architecture decision, life decision |
| `insight` | any | AI-generated cross-project connection |
| `contact` | area | A person or organization |
| `place` | resource | Location worth remembering |

### Relation Types

| Type | Direction | Meaning |
|---|---|---|
| `implements` | rule → decision/task | This task implements this business rule |
| `solves` | decision → problem/note | This decision solves this problem |
| `contradicts` | any → any | These two things conflict |
| `relates` | any → any | General semantic relationship |
| `inspired_by` | any → any | This was created because of that |
| `blocks` | task → task | This task blocks that one |
| `part_of` | any → project | This belongs to this project |

### Metadata by Node Type

```jsonc
// media (movie, book)
{ "genre": "sci-fi", "year": 2024, "creator": "Denis Villeneuve", "format": "movie" }

// event (appointment)
{ "scheduled_at": "2026-04-15T14:00:00Z", "location": "Dr. Smith's office", "recurrence": null }

// rule (business rule)
{ "project": "payments", "language": "go", "enforced_by": "validateTransfer()" }

// decision (architecture decision)
{ "project": "payments", "options_considered": ["REST", "gRPC"], "chosen": "REST", "reason": "..." }
```

---

## API Endpoints

All under `/api/v1/nodes` and `/api/v1/edges`.

### Nodes

```
POST   /api/v1/nodes              Create a node
GET    /api/v1/nodes/:id          Get a node with its edges
GET    /api/v1/nodes              List nodes (filter: type, para, project_id)
PUT    /api/v1/nodes/:id          Update a node
DELETE /api/v1/nodes/:id          Soft delete (sets deleted_at)
```

### Edges

```
POST   /api/v1/edges              Create an edge between two nodes
GET    /api/v1/nodes/:id/edges    Get all edges for a node
DELETE /api/v1/edges/:id          Remove an edge
```

### Graph traversal

```
GET /api/v1/nodes/:id/graph?depth=2
```

Returns the node and all nodes reachable within `depth` hops, with their edges. Used for context loading and visualization.

---

## AI Integration

When a new node is created:
1. AI generates embedding → stored in `embedding`
2. AI sets initial `review_stage = 0` and `next_review_at = now() + 2 weeks`
3. AI creates initial edges if obvious relationships exist (e.g., a new rule node in a known project gets `part_of` edge automatically)

When the insight linker (T-029) runs:
- It creates `insight` nodes and `relates` / `solves` edges with `created_by = 'insight-linker'` and `confidence < 1.0`

---

## Acceptance Criteria

- [ ] `nodes` and `edges` tables created via migration
- [ ] All CRUD endpoints for nodes
- [ ] All CRUD endpoints for edges
- [ ] GET /api/v1/nodes/:id/graph returns traversal up to depth N
- [ ] Node creation triggers embedding generation (async via Asynq)
- [ ] Soft delete: deleted nodes excluded from all queries, edges cascade-hidden
- [ ] Archived nodes excluded from active queries but retrievable
- [ ] Filter nodes by type, para, project_id
- [ ] Unique constraint on (from_node_id, to_node_id, relation_type) — no duplicate edges
- [ ] Unit tests for NodeService and EdgeService
- [ ] Integration tests for all endpoints
- [ ] 90% test coverage

---

## Dependencies

- T-004 (PostgreSQL + pgvector)
- T-007 (Auth — user_id on all nodes)
- T-020 (AI provider — embedding generation)
- T-036 (Relevance decay — review_stage fields)

## Notes

- Existing `notes` and `tasks` tables (T-010, T-030) should be migrated to the `nodes` table, or kept as domain tables with a `node_id` foreign key. Decision: keep domain tables, add `node_id` FK to each — avoids a big table with all columns nullable.
- `project_id` on nodes is a self-referential FK — a project is itself a node of type `project`.
- The graph is per-user. There is no shared graph.
