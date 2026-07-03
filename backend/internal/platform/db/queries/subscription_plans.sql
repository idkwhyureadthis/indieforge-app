-- name: UpsertSubscriptionPlan :one
INSERT INTO subscription_plans (id, developer_id, name, price, period, benefits, chat_link)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (developer_id) DO UPDATE SET
    name      = EXCLUDED.name,
    price     = EXCLUDED.price,
    period    = EXCLUDED.period,
    benefits  = EXCLUDED.benefits,
    chat_link = EXCLUDED.chat_link
RETURNING *;

-- name: GetPlanByDeveloper :one
SELECT * FROM subscription_plans WHERE developer_id = $1;

-- name: GetPlanByID :one
SELECT * FROM subscription_plans WHERE id = $1;

-- name: AddGameToPlan :exec
INSERT INTO subscription_plan_games (plan_id, game_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveGameFromPlan :exec
DELETE FROM subscription_plan_games WHERE plan_id = $1 AND game_id = $2;

-- name: ListPlanGames :many
SELECT g.* FROM games g
JOIN subscription_plan_games spg ON spg.game_id = g.id
WHERE spg.plan_id = $1
ORDER BY g.created_at DESC;

-- name: ListPlanGameIDs :many
SELECT game_id FROM subscription_plan_games WHERE plan_id = $1;

-- name: GetPlanForGameKey :one
SELECT sp.* FROM subscription_plans sp
JOIN games g ON g.developer_id = sp.developer_id
WHERE (g.id = $1 OR g.slug = $1) AND sp.active = TRUE
LIMIT 1;

-- name: SetPaymentPlanID :exec
UPDATE payments SET plan_id = $2 WHERE id = $1;
