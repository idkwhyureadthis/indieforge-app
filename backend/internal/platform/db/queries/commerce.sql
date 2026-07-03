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

-- name: DeleteOwnership :exec
DELETE FROM ownerships WHERE user_id = $1 AND game_id = $2;

-- name: SetSubscriptionRenewalInfo :exec
UPDATE subscriptions
SET expires_at = $2, payment_method_id = $3
WHERE id = $1;

-- name: ExtendSubscription :exec
UPDATE subscriptions
SET expires_at = $2
WHERE id = $1;

-- name: DeactivateSubscription :exec
UPDATE subscriptions SET active = FALSE WHERE id = $1;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions WHERE id = $1;

-- name: ListExpiringSubscriptions :many
SELECT * FROM subscriptions
WHERE active = TRUE
  AND expires_at IS NOT NULL
  AND payment_method_id != ''
  AND expires_at <= $1
ORDER BY expires_at;

-- name: SetPaymentSubID :exec
UPDATE payments SET sub_id = $2 WHERE id = $1;

-- name: SetPaymentMethodID :exec
UPDATE payments SET payment_method_id = $2 WHERE id = $1;

-- name: ListSubscriptionsWithExpiry :many
SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY started_at DESC;

-- name: GetUserSubscriptionStatus :one
SELECT s.active, s.expires_at
FROM subscriptions s
JOIN games g ON g.id = s.game_id
WHERE s.user_id = $1
  AND (g.id = $2 OR g.slug = $2)
  AND s.active = TRUE
LIMIT 1;

-- name: GetGameIDBySlugOrID :one
SELECT id FROM games WHERE id = $1 OR slug = $1;
