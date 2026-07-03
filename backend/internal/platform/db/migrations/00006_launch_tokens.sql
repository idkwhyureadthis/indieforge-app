-- +goose Up

-- Short-lived tokens generated on the game page so a downloadable game can
-- identify the player without embedding any credentials in the game binary.
CREATE TABLE launch_tokens (
    token_hash  TEXT        PRIMARY KEY,          -- SHA-256 hex
    user_id     TEXT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id     TEXT        NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '15 minutes'
);

CREATE INDEX idx_launch_tokens_expires ON launch_tokens(expires_at);

-- +goose Down
DROP TABLE IF EXISTS launch_tokens;
