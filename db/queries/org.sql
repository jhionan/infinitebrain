-- Org queries for sqlc code generation.

-- name: FindOrgByID :one
SELECT id, name, slug, plan, max_members, settings, phi_enabled, created_at, updated_at
FROM orgs
WHERE id = $1
  AND deleted_at IS NULL;

-- name: FindOrgBySlug :one
SELECT id, name, slug, plan, max_members, settings, phi_enabled, created_at, updated_at
FROM orgs
WHERE slug = $1
  AND deleted_at IS NULL;

-- name: UpdateOrg :one
UPDATE orgs
SET name       = $2,
    settings   = $3,
    updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, name, slug, plan, max_members, settings, phi_enabled, created_at, updated_at;

-- name: SoftDeleteOrg :exec
UPDATE orgs
SET deleted_at = now(), updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, user_id, role, invited_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id, user_id) DO NOTHING;

-- name: FindOrgMember :one
SELECT org_id, user_id, role, invited_by, joined_at
FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- name: ListOrgMembers :many
SELECT om.org_id, om.user_id, om.role, om.invited_by, om.joined_at,
       u.email, u.display_name
FROM org_members om
JOIN users u ON u.id = om.user_id AND u.deleted_at IS NULL
WHERE om.org_id = $1
ORDER BY om.joined_at;

-- name: UpdateOrgMemberRole :exec
UPDATE org_members
SET role = $3
WHERE org_id = $1 AND user_id = $2;

-- name: RemoveOrgMember :exec
DELETE FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- name: CountOrgMembers :one
SELECT COUNT(*) FROM org_members WHERE org_id = $1;
