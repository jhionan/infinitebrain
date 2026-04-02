-- db/queries/rbac.sql
-- RBAC queries: audit_log, org_invites, GetUserOrgs.

-- name: InsertAuditLog :exec
INSERT INTO audit_log (org_id, actor_id, action, target_type, target_id, before, after, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLog :many
SELECT id, org_id, actor_id, action, target_type, target_id, before, after, ip, created_at
FROM audit_log
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateOrgInvite :one
INSERT INTO org_invites (org_id, email, role, invited_by, token)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, org_id, email, role, invited_by, token, expires_at, accepted_at, created_at;

-- name: FindOrgInviteByToken :one
SELECT id, org_id, email, role, invited_by, token, expires_at, accepted_at, created_at
FROM org_invites
WHERE token = $1
  AND expires_at > now()
  AND accepted_at IS NULL;

-- name: AcceptOrgInvite :exec
UPDATE org_invites
SET accepted_at = now()
WHERE id = $1;

-- name: GetUserOrgs :many
SELECT o.id, o.name, o.slug, o.plan, om.role
FROM org_members om
JOIN orgs o ON o.id = om.org_id AND o.deleted_at IS NULL
WHERE om.user_id = $1
ORDER BY om.joined_at;
