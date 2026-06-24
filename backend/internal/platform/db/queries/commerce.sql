-- name: CreateOwnership :one
INSERT INTO ownerships (id, user_id, game_id, type, price, gifted_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, game_id) DO NOTHING
RETURNING *;

-- name: GetOwnership :one
SELECT * FROM ownerships WHERE user_id = $1 AND game_id = $2;

-- name: ListOwnershipsByUser :many
SELECT * FROM ownerships WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateSubscription :one
INSERT INTO subscriptions (id, user_id, game_id, developer_id, price)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetActiveSubscription :one
SELECT * FROM subscriptions WHERE user_id = $1 AND game_id = $2 AND active = TRUE;

-- name: ListSubscriptionsByUser :many
SELECT * FROM subscriptions WHERE user_id = $1 AND active = TRUE ORDER BY started_at DESC;

-- name: CreatePayment :one
INSERT INTO payments (id, yk_id, user_id, game_id, kind, amount, commission_percent, commission_amount, status, friend_username)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetPaymentByYkID :one
SELECT * FROM payments WHERE yk_id = $1;

-- name: SetPaymentYkID :exec
UPDATE payments SET yk_id = $2 WHERE id = $1;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = $2 WHERE id = $1;
