-- Note queries for sqlc code generation.
-- Notes are nodes rows where type = 'note'.
-- source and status live in the metadata JSONB column.

-- name: FindDefaultOrgUnit :one
SELECT id FROM org_units
WHERE org_id = $1
  AND unit_type = 'root'
  AND deleted_at IS NULL
LIMIT 1;

-- name: CreateNote :one
INSERT INTO nodes (org_id, user_id, unit_id, type, title, content, tags, visibility, metadata)
VALUES ($1, $2, $3, 'note', $4, $5, $6, $7, $8)
RETURNING id, org_id, user_id, unit_id, title, content, para, project_id,
          tags, visibility, is_phi, metadata, created_at, updated_at, archived_at;

-- name: FindNoteByID :one
SELECT id, org_id, user_id, unit_id, title, content, para, project_id,
       tags, visibility, is_phi, metadata, created_at, updated_at, archived_at
FROM nodes
WHERE id = $1
  AND org_id = $2
  AND type = 'note'
  AND deleted_at IS NULL;

-- name: ListNotes :many
SELECT id, org_id, user_id, unit_id, title, content, para, project_id,
       tags, visibility, is_phi, metadata, created_at, updated_at, archived_at
FROM nodes
WHERE org_id = $1
  AND user_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $3 OFFSET $4;

-- name: CountNotes :one
SELECT COUNT(*) FROM nodes
WHERE org_id = $1
  AND user_id = $2
  AND type = 'note'
  AND deleted_at IS NULL;

-- name: ListInboxNotes :many
SELECT id, org_id, user_id, unit_id, title, content, para, project_id,
       tags, visibility, is_phi, metadata, created_at, updated_at, archived_at
FROM nodes
WHERE org_id = $1
  AND user_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
  AND metadata->>'status' = 'inbox'
ORDER BY created_at DESC, id DESC
LIMIT $3 OFFSET $4;

-- name: CountInboxNotes :one
SELECT COUNT(*) FROM nodes
WHERE org_id = $1
  AND user_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
  AND metadata->>'status' = 'inbox';

-- name: UpdateNote :one
UPDATE nodes
SET title      = $3,
    content    = $4,
    tags       = $5,
    metadata   = metadata || $6::jsonb,
    updated_at = now()
WHERE id = $1
  AND org_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
RETURNING id, org_id, user_id, unit_id, title, content, para, project_id,
          tags, visibility, is_phi, metadata, created_at, updated_at, archived_at;

-- name: SoftDeleteNote :one
UPDATE nodes
SET deleted_at = now(), updated_at = now()
WHERE id = $1
  AND org_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
RETURNING id;

-- name: ArchiveNote :one
UPDATE nodes
SET archived_at = now(),
    metadata    = metadata || '{"status":"archived"}'::jsonb,
    updated_at  = now()
WHERE id = $1
  AND org_id = $2
  AND type = 'note'
  AND deleted_at IS NULL
RETURNING id, org_id, user_id, unit_id, title, content, para, project_id,
          tags, visibility, is_phi, metadata, created_at, updated_at, archived_at;
