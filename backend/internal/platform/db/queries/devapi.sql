-- name: CreateAPIKey :one
INSERT INTO developer_api_keys (id, developer_id, name, key_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM developer_api_keys
WHERE key_hash = $1 AND revoked = FALSE;

-- name: ListAPIKeysByDeveloper :many
SELECT id, developer_id, name, created_at, last_used_at, revoked
FROM developer_api_keys
WHERE developer_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIKey :exec
UPDATE developer_api_keys
SET revoked = TRUE
WHERE id = $1 AND developer_id = $2;

-- name: TouchAPIKey :exec
UPDATE developer_api_keys
SET last_used_at = now()
WHERE id = $1;

-- name: GetSubscriptionForVerify :one
SELECT s.active, s.expires_at
FROM subscriptions s
JOIN games g ON g.id = s.game_id
WHERE s.user_id = $1
  AND s.game_id = $2
  AND g.developer_id = $3
  AND s.active = TRUE
LIMIT 1;

-- name: GetGameDeveloperID :one
SELECT developer_id FROM games WHERE id = $1 OR slug = $1;

-- name: CreateLaunchToken :exec
INSERT INTO launch_tokens (token_hash, user_id, game_id)
VALUES ($1, $2, $3);

-- name: GetLaunchToken :one
SELECT lt.user_id, lt.game_id, lt.expires_at,
       COALESCE(s.active, FALSE) AS subscribed,
       s.expires_at              AS sub_expires_at
FROM launch_tokens lt
LEFT JOIN subscriptions s
    ON s.user_id = lt.user_id AND s.game_id = lt.game_id AND s.active = TRUE
WHERE lt.token_hash = $1 AND lt.expires_at > now()
LIMIT 1;

-- name: DeleteLaunchToken :exec
DELETE FROM launch_tokens WHERE token_hash = $1;

-- name: PurgeLaunchTokens :exec
DELETE FROM launch_tokens WHERE expires_at < now();
