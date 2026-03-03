# T-172 — Project Management Connectors

## Overview

Infinite Brain is the source of truth. External PM tools (Jira, Asana, Linear, GitHub Projects)
are synchronized views. IB can import tasks from these tools as nodes, sync status changes
bidirectionally, and surface PM tool context alongside IB's knowledge graph.

**Security note**: All content imported from PM tools passes through PromptGuard (T-177)
before AI processing. Jira ticket descriptions, GitHub issue bodies, and Asana task notes
are all attacker-accessible and must be treated as untrusted external content.

---

## The Integration Model

```
External PM Tool
    │
    ├── Webhooks (real-time push): task created/updated/completed
    │       ↓ PromptGuard sanitization
    │       ↓ Node created/updated in IB
    │
    └── Polling API (initial import + gap fill)
            ↓ PromptGuard sanitization
            ↓ Bulk node creation in IB

IB (source of truth)
    │
    └── Outbound sync: when IB task status changes → push to PM tool via their API
```

IB never merges identity: an IB node created from a Jira ticket retains `external_id` and
`external_source` fields. Changes in IB sync outbound. Changes in the PM tool sync inbound.
Conflicts are resolved by last-write-wins with a human-review flag for significant changes.

---

## The Connector Interface

```go
// internal/integrations/pm/connector.go

type PMConnector interface {
    // Identity
    Name() string  // "jira" | "asana" | "linear" | "github_projects"

    // Pull: import tasks from external tool into IB
    ListTasks(ctx context.Context, filter TaskFilter) ([]ExternalTask, error)
    GetTask(ctx context.Context, externalID string) (*ExternalTask, error)

    // Push: sync IB task changes to external tool
    UpdateTaskStatus(ctx context.Context, externalID string, status TaskStatus) error
    CreateTask(ctx context.Context, task NewTaskRequest) (string, error) // returns externalID

    // Webhooks: verify + parse inbound events
    VerifyWebhook(payload []byte, signature string, secret string) error
    ParseWebhookEvent(payload []byte) (*WebhookEvent, error)
}

type ExternalTask struct {
    ExternalID   string
    Source       string          // "jira" | "asana" | etc.
    Title        string
    Description  string          // UNTRUSTED — passed through PromptGuard before AI
    Status       string          // raw status from external system
    Assignee     *string
    DueDate      *time.Time
    Labels       []string
    ProjectKey   string
    URL          string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type WebhookEvent struct {
    EventType    string          // "task.created" | "task.updated" | "task.deleted"
    ExternalID   string
    Task         ExternalTask
    RawPayload   json.RawMessage
}
```

---

## Node Representation

Imported PM tasks become IB nodes of `type = "task"` with external identity preserved:

```go
// metadata JSON on the node:
{
    "external_source": "jira",
    "external_id": "PROJ-123",
    "external_url": "https://company.atlassian.net/browse/PROJ-123",
    "external_status": "In Progress",
    "sync_direction": "bidirectional",  // or "import_only"
    "last_synced_at": "2026-04-01T10:00:00Z"
}
```

Status mapping is connector-specific — each PM tool has different status names.
IB normalizes to its own status model: `inbox | active | blocked | done | archived`.

---

## Supported Connectors (Phase 1)

### Jira (Atlassian)
- Auth: OAuth 2.0 (Jira Cloud) or API token (Jira Server)
- Webhooks: Jira webhook system with secret validation
- Import: JQL query filter (e.g. `project = PROJ AND status != Done`)
- Sync: issue status + assignee + due date
- T-173

### Linear
- Auth: OAuth 2.0 or personal API key
- Webhooks: Linear webhook with signature verification
- Import: GraphQL API filter by team/cycle/label
- Sync: issue status + priority + due date
- T-176

### GitHub Projects
- Auth: GitHub App or personal token
- Webhooks: GitHub webhook with HMAC-SHA256 signature
- Import: GraphQL API (Projects V2)
- Sync: item status + custom fields
- Extends T-073
- T-175

### Asana
- Auth: OAuth 2.0 or personal access token
- Webhooks: Asana webhook with X-Hook-Secret validation
- Import: task search API with project filter
- Sync: task completion + assignee + due date
- T-174

---

## Security: Webhook Verification

Every inbound webhook is verified before processing.
Unverified payloads are rejected — not processed, not logged to knowledge graph.

```go
// pkg/integrations/webhook.go

func VerifyJiraWebhook(payload []byte, signature, secret string) error {
    // Jira uses HMAC-SHA256
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    if !hmac.Equal([]byte(expected), []byte(signature)) {
        return apperrors.ErrUnauthorized.Wrap("invalid webhook signature")
    }
    return nil
}

func VerifyGitHubWebhook(payload []byte, signature, secret string) error {
    // GitHub uses X-Hub-Signature-256: sha256=<hex>
    sig := strings.TrimPrefix(signature, "sha256=")
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    if !hmac.Equal([]byte(expected), []byte(sig)) {
        return apperrors.ErrUnauthorized.Wrap("invalid webhook signature")
    }
    return nil
}
```

---

## Prompt Injection in PM Content

Jira descriptions, GitHub issue bodies, and Asana task notes are written by external users.
An attacker with Jira access can craft a ticket description that injects prompts into IB.

Mitigation:
1. All `ExternalTask.Description` fields pass through `PromptGuard.Sanitize` with `TrustLevelKnown`
2. Title fields are treated as plain text only (no AI processing beyond embedding)
3. The AI call that processes PM content uses the classification prompt template with
   explicit content-instruction boundary (T-177)
4. Output is validated against schema before node creation (T-177)

---

## Acceptance Criteria

- [ ] `PMConnector` interface with all methods above
- [ ] Factory: `pm.NewConnector(source string, config ConnectorConfig) PMConnector`
- [ ] Jira connector (T-173): OAuth, webhooks, import, bidirectional sync
- [ ] Linear connector (T-176): OAuth, webhooks, import, bidirectional sync
- [ ] GitHub Projects connector (T-175): App auth, webhooks, import, sync
- [ ] Asana connector (T-174): OAuth, webhooks, import, sync
- [ ] All webhook handlers verify signature before processing
- [ ] All imported content passes through PromptGuard before AI
- [ ] `external_source` + `external_id` preserved on nodes
- [ ] Status mapping: PM tool statuses → IB statuses
- [ ] Conflict resolution: last-write-wins + human review flag for significant changes
- [ ] 90% test coverage; webhook signature verification at 100%

## Dependencies

- T-177 (PromptGuard — all imported content sanitized before AI)
- T-030 (task CRUD — imported tasks become IB task nodes)
- T-028 (knowledge graph — nodes with external identity)
- T-073 (GitHub integration — Projects connector extends this)
- T-120 (event sourcing — sync events)
