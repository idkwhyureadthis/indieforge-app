-- name: CreateReport :one
INSERT INTO reports (id, reporter_id, target_type, target_id, reason, details)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetReport :one
SELECT * FROM reports WHERE id = $1;

-- name: ListReports :many
SELECT * FROM reports WHERE status = $1 ORDER BY created_at DESC;

-- name: ListAllReports :many
SELECT * FROM reports ORDER BY created_at DESC;

-- name: ResolveReport :exec
UPDATE reports
SET status = $2, resolution = $3, handled_by = $4, resolved_at = now()
WHERE id = $1;
