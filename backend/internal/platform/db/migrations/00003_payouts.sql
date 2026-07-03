-- +goose Up

CREATE TABLE payouts (
    id           TEXT PRIMARY KEY,
    developer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount       INTEGER NOT NULL,           -- kopecks, must be > 0
    status       TEXT NOT NULL DEFAULT 'pending',  -- pending | paid | rejected
    note         TEXT NOT NULL DEFAULT '',   -- admin note on rejection / confirmation
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payouts_developer ON payouts(developer_id);
CREATE INDEX idx_payouts_status    ON payouts(status);

-- +goose Down

DROP TABLE IF EXISTS payouts;
