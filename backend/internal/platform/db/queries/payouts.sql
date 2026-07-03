-- name: CreatePayout :one
INSERT INTO payouts (id, developer_id, amount)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPayoutByID :one
SELECT * FROM payouts WHERE id = $1;

-- name: ListPayoutsByDeveloper :many
SELECT * FROM payouts WHERE developer_id = $1 ORDER BY created_at DESC;

-- name: ListAllPayouts :many
SELECT po.*, u.username AS developer_username
FROM payouts po
JOIN users u ON u.id = po.developer_id
ORDER BY po.created_at DESC;

-- name: UpdatePayoutStatus :one
UPDATE payouts
SET status = $2, note = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- Developer earnings: sum of (amount - commission) for all succeeded payments on their games.
-- name: GetDeveloperEarnings :one
SELECT COALESCE(SUM(p.amount - p.commission_amount), 0)::BIGINT AS total_earned
FROM payments p
JOIN games g ON g.id = p.game_id
WHERE g.developer_id = $1
  AND p.status = 'succeeded';

-- Total already requested (pending + paid); rejected ones don't count.
-- name: GetDeveloperPayoutsTotal :one
SELECT COALESCE(SUM(amount), 0)::BIGINT AS total_requested
FROM payouts
WHERE developer_id = $1
  AND status != 'rejected';
