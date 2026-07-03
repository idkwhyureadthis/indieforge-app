-- +goose Up

CREATE TABLE subscription_plans (
    id           TEXT PRIMARY KEY,
    developer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL DEFAULT 'Creator Pack',
    price        INT  NOT NULL DEFAULT 0,
    period       TEXT NOT NULL DEFAULT 'month',
    benefits     TEXT[] NOT NULL DEFAULT '{}',
    chat_link    TEXT NOT NULL DEFAULT '',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (developer_id)
);

CREATE TABLE subscription_plan_games (
    plan_id TEXT NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, game_id)
);

-- Track which plan a payment was for (NULL for regular game payments).
ALTER TABLE payments ADD COLUMN plan_id TEXT REFERENCES subscription_plans(id);

CREATE INDEX idx_sub_plans_developer ON subscription_plans(developer_id);
CREATE INDEX idx_payments_plan ON payments(plan_id) WHERE plan_id IS NOT NULL;

-- +goose Down

ALTER TABLE payments DROP COLUMN plan_id;
DROP TABLE IF EXISTS subscription_plan_games;
DROP TABLE IF EXISTS subscription_plans;
