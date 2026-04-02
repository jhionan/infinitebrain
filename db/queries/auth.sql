-- Auth queries for sqlc code generation.

-- name: FindUserByEmail :one
SELECT id, org_id, email, display_name, role, password_hash, pepper_version, created_at, updated_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: FindUserByID :one
SELECT id, org_id, email, display_name, role, password_hash, pepper_version, created_at, updated_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;

-- name: CreateOrg :one
INSERT INTO orgs (name, slug, plan)
VALUES ($1, $2, $3)
RETURNING id, name, slug, plan, max_members, settings, phi_enabled, created_at, updated_at;

-- name: CreateUser :one
INSERT INTO users (org_id, email, display_name, role, password_hash, pepper_version)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, org_id, email, display_name, role, password_hash, pepper_version, created_at, updated_at;

-- name: UpdateUserLastActive :exec
UPDATE users
SET last_active_at = now(), updated_at = now()
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, org_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, org_id, token_hash, expires_at, created_at;

-- name: FindSessionByTokenHash :one
SELECT id, user_id, org_id, token_hash, expires_at, created_at
FROM sessions
WHERE token_hash = $1
  AND expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < now();

-- name: CreateOrgUnit :one
INSERT INTO org_units (org_id, name, unit_type)
VALUES ($1, $2, $3)
RETURNING id, org_id, parent_unit_id, name, unit_type, created_at, updated_at, deleted_at;
