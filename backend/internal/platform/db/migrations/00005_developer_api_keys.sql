-- +goose Up

CREATE TABLE developer_api_keys (
    id            TEXT PRIMARY KEY,
    developer_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL DEFAULT '',
    key_hash      TEXT NOT NULL UNIQUE,   -- SHA-256 hex of the plaintext key
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at  TIMESTAMPTZ,
    revoked       BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_api_keys_developer ON developer_api_keys(developer_id);

-- +goose Down

DROP TABLE IF EXISTS developer_api_keys;
