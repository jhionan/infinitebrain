-- name: InsertHoneypotHit :exec
INSERT INTO honeypot_hits (ip, path, method, user_agent, headers, body)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountHoneypotHitsLast24h :one
SELECT COUNT(*)::INT AS hit_count
FROM honeypot_hits
WHERE ip = $1
  AND created_at >= now() - INTERVAL '24 hours';

-- name: ListRecentHoneypotHits :many
SELECT id, ip, path, method, user_agent, headers, body, hit_count, blocked_at, created_at
FROM honeypot_hits
ORDER BY created_at DESC
LIMIT $1;

-- name: UpsertBlockedIP :exec
INSERT INTO blocked_ips (ip, reason, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (ip) DO UPDATE
    SET reason     = EXCLUDED.reason,
        expires_at = EXCLUDED.expires_at;

-- name: GetBlockedIP :one
SELECT ip, reason, expires_at, created_at
FROM blocked_ips
WHERE ip = $1;

-- name: DeleteBlockedIP :exec
DELETE FROM blocked_ips
WHERE ip = $1;

-- name: ListBlockedIPs :many
SELECT ip, reason, expires_at, created_at
FROM blocked_ips
ORDER BY created_at DESC;

-- name: DeleteExpiredBlockedIPs :exec
DELETE FROM blocked_ips
WHERE expires_at IS NOT NULL
  AND expires_at < now();
