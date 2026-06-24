-- name: CreateGame :one
INSERT INTO games (
    id, slug, title, tagline, description, genre, tags,
    developer_id, developer_name, cover_image, screenshots,
    has_browser_build, browser_build_url,
    has_download_build, download_object_key, download_file_name, download_size_mb, download_platforms,
    supports_multiplayer, pricing_model, price, friend_pack_discount,
    sub_enabled, sub_price, sub_benefits, sub_chat_link,
    demo_enabled, demo_starts_at, demo_ends_at,
    theme, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11,
    $12, $13,
    $14, $15, $16, $17, $18,
    $19, $20, $21, $22,
    $23, $24, $25, $26,
    $27, $28, $29,
    $30, $31
)
RETURNING *;

-- name: GetGameByID :one
SELECT * FROM games WHERE id = $1;

-- name: GetGameBySlug :one
SELECT * FROM games WHERE slug = $1;

-- name: ListPublishedGames :many
SELECT * FROM games WHERE status = 'published' ORDER BY created_at DESC;

-- name: ListGamesByDeveloper :many
SELECT * FROM games WHERE developer_id = $1 ORDER BY created_at DESC;

-- name: ListNewest :many
SELECT * FROM games WHERE status = 'published' ORDER BY created_at DESC LIMIT $1;

-- name: ListTrending :many
SELECT * FROM games
WHERE status = 'published' AND trending_score > 0
ORDER BY trending_score DESC
LIMIT $1;

-- name: ListPopular :many
SELECT * FROM games
WHERE status = 'published'
ORDER BY
    (SELECT count(*) FROM ownerships o WHERE o.game_id = games.id)
    + (SELECT count(*) FROM subscriptions s WHERE s.game_id = games.id AND s.active) DESC,
    created_at DESC
LIMIT $1;

-- name: OwnerCounts :many
SELECT game_id, count(*)::int AS n FROM ownerships GROUP BY game_id;

-- name: SubscriberCounts :many
SELECT game_id, count(*)::int AS n FROM subscriptions WHERE active GROUP BY game_id;

-- name: CountOwnersByGame :one
SELECT count(*)::int FROM ownerships WHERE game_id = $1;

-- name: CountSubscribersByGame :one
SELECT count(*)::int FROM subscriptions WHERE game_id = $1 AND active;

-- name: SlugExists :one
SELECT EXISTS (SELECT 1 FROM games WHERE slug = $1);

-- name: IncrementPlays :exec
UPDATE games SET plays = plays + 1 WHERE id = $1;

-- name: SetGameStatus :exec
UPDATE games SET status = $2 WHERE id = $1;

-- name: InsertGameEvent :exec
INSERT INTO game_events (id, game_id, type) VALUES ($1, $2, $3);

-- name: RecomputeTrendingScores :exec
UPDATE games g SET trending_score = COALESCE((
    SELECT sum(
        (CASE e.type WHEN 'acquire' THEN 5.0 WHEN 'play' THEN 2.0 ELSE 1.0 END)
        * exp(- extract(epoch FROM (now() - e.created_at)) / 604800.0)
    )
    FROM game_events e
    WHERE e.game_id = g.id AND e.created_at > now() - interval '7 days'
), 0);
