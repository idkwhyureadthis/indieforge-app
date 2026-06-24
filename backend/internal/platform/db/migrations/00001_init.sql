-- +goose Up
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',   -- user | moderator | admin
    is_developer  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE games (
    id                   TEXT PRIMARY KEY,
    slug                 TEXT NOT NULL UNIQUE,
    title                TEXT NOT NULL,
    tagline              TEXT NOT NULL DEFAULT '',
    description          TEXT NOT NULL DEFAULT '',
    genre                TEXT NOT NULL DEFAULT 'Other',
    tags                 TEXT[] NOT NULL DEFAULT '{}',
    developer_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    developer_name       TEXT NOT NULL,
    cover_image          TEXT NOT NULL DEFAULT '',       -- public object URL
    screenshots          TEXT[] NOT NULL DEFAULT '{}',   -- public object URLs
    has_browser_build    BOOLEAN NOT NULL DEFAULT FALSE,
    browser_build_url    TEXT NOT NULL DEFAULT '',       -- public URL of extracted index.html
    has_download_build   BOOLEAN NOT NULL DEFAULT FALSE,
    download_object_key  TEXT NOT NULL DEFAULT '',       -- private S3 key
    download_file_name   TEXT NOT NULL DEFAULT '',
    download_size_mb     INTEGER NOT NULL DEFAULT 0,
    download_platforms   TEXT[] NOT NULL DEFAULT '{}',
    supports_multiplayer BOOLEAN NOT NULL DEFAULT FALSE,
    pricing_model        TEXT NOT NULL DEFAULT 'free',
    price                INTEGER NOT NULL DEFAULT 0,
    friend_pack_discount INTEGER NOT NULL DEFAULT 0,
    sub_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    sub_price            INTEGER NOT NULL DEFAULT 0,
    sub_benefits         TEXT[] NOT NULL DEFAULT '{}',
    sub_chat_link        TEXT NOT NULL DEFAULT '',       -- subscriber-only perk
    demo_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    demo_starts_at       TIMESTAMPTZ,
    demo_ends_at         TIMESTAMPTZ,
    theme                JSONB NOT NULL DEFAULT '{}',    -- {accent, accent2, background, layout, cardShape}
    status               TEXT NOT NULL DEFAULT 'published', -- published | draft | hidden | removed
    plays                INTEGER NOT NULL DEFAULT 0,
    trending_score       DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_games_developer ON games(developer_id);
CREATE INDEX idx_games_status ON games(status);
CREATE INDEX idx_games_trending ON games(trending_score DESC);

-- Append-only activity log powering the trending score.
CREATE TABLE game_events (
    id         TEXT PRIMARY KEY,
    game_id    TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,   -- view | play | acquire
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_game_events_game_time ON game_events(game_id, created_at);
CREATE INDEX idx_game_events_time ON game_events(created_at);

CREATE TABLE ownerships (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id    TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    price      INTEGER NOT NULL DEFAULT 0,
    gifted_by  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, game_id)
);

CREATE TABLE subscriptions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id      TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    developer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price        INTEGER NOT NULL DEFAULT 0,
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id                 TEXT PRIMARY KEY,
    yk_id              TEXT NOT NULL DEFAULT '',
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id            TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    kind               TEXT NOT NULL,
    amount             INTEGER NOT NULL DEFAULT 0,
    commission_percent INTEGER NOT NULL DEFAULT 0,   -- snapshot at payment time
    commission_amount  INTEGER NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'pending',
    friend_username    TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_yk ON payments(yk_id);

CREATE TABLE reports (
    id          TEXT PRIMARY KEY,
    reporter_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL DEFAULT 'game',
    target_id   TEXT NOT NULL,
    reason      TEXT NOT NULL,   -- inappropriate | copyright | broken | scam | other
    details     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'open',  -- open | resolved | dismissed
    resolution  TEXT NOT NULL DEFAULT '',
    handled_by  TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_status ON reports(status);

-- Singleton runtime settings (id is always TRUE so only one row can exist).
CREATE TABLE settings (
    id                 BOOLEAN PRIMARY KEY DEFAULT TRUE,
    commission_percent INTEGER NOT NULL DEFAULT 10,
    trending_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    popular_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT settings_singleton CHECK (id)
);

INSERT INTO settings (id) VALUES (TRUE);

-- +goose Down
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS ownerships;
DROP TABLE IF EXISTS game_events;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
