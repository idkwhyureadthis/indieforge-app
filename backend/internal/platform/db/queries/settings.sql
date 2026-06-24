-- name: GetSettings :one
SELECT * FROM settings WHERE id = TRUE;

-- name: UpdateSettings :one
UPDATE settings
SET commission_percent = $1,
    trending_enabled = $2,
    popular_enabled = $3,
    updated_at = now()
WHERE id = TRUE
RETURNING *;
