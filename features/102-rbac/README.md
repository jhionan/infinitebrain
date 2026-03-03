# T-102 — RBAC (Role-Based Access Control)

## Overview

Four roles control what org members can do: owner, admin, editor, viewer. Permissions are checked at the service layer before every mutation. Zitadel roles are synced into Infinite Brain claims — enterprise customers can manage roles via their existing identity infrastructure.

---

## Roles

| Role | Who | What they can do |
|---|---|---|
| `owner` | Org creator, 1 per org | Everything + delete org + billing + transfer ownership |
| `admin` | Trusted managers | Everything except billing and org deletion |
| `editor` | Regular team members | Create/edit/delete own content; read all org content |
| `viewer` | Read-only guests, clients | Read all org content; no mutations |

---

## Permission Matrix

| Action | owner | admin | editor | viewer |
|---|:---:|:---:|:---:|:---:|
| Read any node | ✓ | ✓ | ✓ | ✓ |
| Create node | ✓ | ✓ | ✓ | — |
| Edit own node | ✓ | ✓ | ✓ | — |
| Edit any node | ✓ | ✓ | — | — |
| Delete own node | ✓ | ✓ | ✓ | — |
| Delete any node | ✓ | ✓ | — | — |
| Manage members | ✓ | ✓ | — | — |
| Change member roles | ✓ | ✓ | — | — |
| View audit log | ✓ | ✓ | — | — |
| Manage org settings | ✓ | ✓ | — | — |
| Billing / plan | ✓ | — | — | — |
| Delete org | ✓ | — | — | — |
| Transfer ownership | ✓ | — | — | — |
| Create API tokens | ✓ | ✓ | ✓ | — |
| Manage honeypot | ✓ | ✓ | — | — |

---

## Implementation

### Permission constants

```go
// internal/auth/permissions.go

type Permission string

const (
    PermReadNode        Permission = "node:read"
    PermCreateNode      Permission = "node:create"
    PermEditOwnNode     Permission = "node:edit:own"
    PermEditAnyNode     Permission = "node:edit:any"
    PermDeleteOwnNode   Permission = "node:delete:own"
    PermDeleteAnyNode   Permission = "node:delete:any"
    PermManageMembers   Permission = "members:manage"
    PermChangeRoles     Permission = "members:roles"
    PermViewAuditLog    Permission = "audit:read"
    PermManageSettings  Permission = "org:settings"
    PermBilling         Permission = "org:billing"
    PermDeleteOrg       Permission = "org:delete"
    PermTransferOwner   Permission = "org:transfer"
    PermCreateToken     Permission = "tokens:create"
    PermManageHoneypot  Permission = "honeypot:manage"
)

var rolePermissions = map[string][]Permission{
    "owner": {
        PermReadNode, PermCreateNode, PermEditOwnNode, PermEditAnyNode,
        PermDeleteOwnNode, PermDeleteAnyNode, PermManageMembers, PermChangeRoles,
        PermViewAuditLog, PermManageSettings, PermBilling, PermDeleteOrg,
        PermTransferOwner, PermCreateToken, PermManageHoneypot,
    },
    "admin": {
        PermReadNode, PermCreateNode, PermEditOwnNode, PermEditAnyNode,
        PermDeleteOwnNode, PermDeleteAnyNode, PermManageMembers, PermChangeRoles,
        PermViewAuditLog, PermManageSettings, PermCreateToken, PermManageHoneypot,
    },
    "editor": {
        PermReadNode, PermCreateNode, PermEditOwnNode,
        PermDeleteOwnNode, PermCreateToken,
    },
    "viewer": {
        PermReadNode,
    },
}

func Can(role string, perm Permission) bool {
    perms, ok := rolePermissions[role]
    if !ok {
        return false
    }
    for _, p := range perms {
        if p == perm {
            return true
        }
    }
    return false
}
```

### Middleware (Huma v2)

```go
// internal/middleware/rbac.go

func Require(perm Permission) func(ctx huma.Context, next func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        claims := auth.ClaimsFromContext(ctx.Context())
        if claims == nil {
            huma.WriteErr(ctx, http.StatusUnauthorized, "authentication required")
            return
        }
        if !Can(claims.Role, perm) {
            huma.WriteErr(ctx, http.StatusForbidden,
                fmt.Sprintf("role '%s' cannot perform '%s'", claims.Role, perm))
            return
        }
        next(ctx)
    }
}
```

Usage in route registration:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodDelete,
    Path:        "/api/v1/nodes/{id}",
    Middlewares: huma.Middlewares{Require(PermDeleteAnyNode)},
}, deleteNodeHandler)
```

### Own vs Any resource check

For `edit:own` / `delete:own`, the service layer checks ownership after the permission middleware passes:

```go
// internal/knowledge/node_service.go

func (s *NodeService) Delete(ctx context.Context, nodeID uuid.UUID) error {
    claims := auth.ClaimsFromContext(ctx)

    node, err := s.repo.Get(ctx, nodeID)
    if err != nil {
        return err
    }

    // Editors can only delete their own nodes
    if !Can(claims.Role, PermDeleteAnyNode) {
        if node.CreatedBy != claims.UserID {
            return ErrForbidden
        }
    }

    return s.repo.Delete(ctx, nodeID)
}
```

---

## Zitadel Role Sync

Zitadel supports custom roles per application. Roles defined in Zitadel flow into Infinite Brain via OIDC claims:

```jsonc
// Zitadel token claims (custom claim)
{
  "sub": "user-uuid",
  "email": "alice@acme.com",
  "urn:zitadel:iam:org:project:roles": {
    "admin": { "org-id": "org-uuid" }
  }
}
```

The `OIDCAuthenticator` extracts the role from this claim:

```go
func (a *OIDCAuthenticator) extractRole(authCtx *oauth.IntrospectionContext, orgID string) string {
    roles := authCtx.GetClaimValue("urn:zitadel:iam:org:project:roles")
    // Parse role map, find role for this org
    // Default to "viewer" if no role found
    return roleForOrg(roles, orgID)
}
```

Enterprise customers manage roles in Zitadel admin console — no code change needed.

---

## Role Management API

```
GET    /api/v1/orgs/:slug/members              List members with roles
POST   /api/v1/orgs/:slug/invites              Invite by email (sets pending role)
PUT    /api/v1/orgs/:slug/members/:user_id     Change member role (admin+ only)
DELETE /api/v1/orgs/:slug/members/:user_id     Remove member (admin+ only)

GET    /api/v1/me/orgs                         List all orgs the current user belongs to
GET    /api/v1/me/permissions                  List current user's permissions in current org
```

### Invite flow

```sql
CREATE TABLE org_invites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'editor',
    invited_by UUID NOT NULL REFERENCES users(id),
    token      TEXT NOT NULL UNIQUE,    -- single-use invite token
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '7 days'),
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Invite token is sent via email. On acceptance, user is added to `org_members` with the specified role.

---

## Audit Log

Every role change and permission-sensitive action is logged:

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id),
    actor_id    UUID NOT NULL REFERENCES users(id),
    action      TEXT NOT NULL,      -- 'node.delete' | 'member.role_change' | 'org.settings_update'
    target_type TEXT,               -- 'node' | 'user' | 'org'
    target_id   UUID,
    before      JSONB,              -- state before change
    after       JSONB,              -- state after change
    ip          INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON audit_log (org_id, created_at DESC);
CREATE INDEX ON audit_log (actor_id);
```

```go
// internal/audit/logger.go

type AuditLogger struct{ repo AuditRepository }

func (l *AuditLogger) Log(ctx context.Context, action string, target interface{}, before, after interface{}) {
    claims := auth.ClaimsFromContext(ctx)
    entry := &AuditEntry{
        OrgID:      claims.OrgID,
        ActorID:    claims.UserID,
        Action:     action,
        Before:     toJSON(before),
        After:      toJSON(after),
        IP:         ipFromContext(ctx),
    }
    // Fire-and-forget — audit log failure should not block the operation
    go l.repo.Insert(context.Background(), entry)
}
```

---

## Acceptance Criteria

- [ ] `Can(role, permission)` returns correct bool for all role/permission combinations
- [ ] `Require(perm)` middleware returns 403 with role + permission in error when denied
- [ ] Own-resource check in service layer blocks editors from deleting others' nodes
- [ ] Zitadel role claim extracted and used when `AUTH_MODE=oidc`
- [ ] Default role `viewer` applied when no role claim present in token
- [ ] Role management endpoints respect permission requirements
- [ ] Invite flow: create invite, accept via token, user added with correct role
- [ ] Invite tokens expire after 7 days
- [ ] Audit log entry created for: node delete, role change, org settings update, member remove
- [ ] `GET /api/v1/me/permissions` returns current user's permission list
- [ ] Unit tests for `Can()` — all 4 roles × all 15 permissions
- [ ] Unit tests for `Require()` middleware
- [ ] Integration test: editor cannot delete another user's node
- [ ] Integration test: viewer cannot create a node
- [ ] 90% test coverage

---

## Dependencies

- T-007 / T-100 (Auth — claims contain role)
- T-101 (Multi-tenancy — org_members.role column)

## Notes

- Owner role cannot be removed from an org — only transferred. Prevents org lockout.
- One user can have different roles in different orgs — role is per-org, stored in `org_members`
- Audit log is append-only — no UPDATE or DELETE policies on the table
- For the personal plan, the single user is always owner — RBAC middleware still runs but always passes
