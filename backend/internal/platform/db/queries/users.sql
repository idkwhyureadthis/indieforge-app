-- name: CreateUser :one
INSERT INTO users (id, username, email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE lower(email) = lower($1);

-- name: GetUserByUsername :one
SELECT * FROM users WHERE lower(username) = lower($1);

-- name: MarkDeveloper :exec
UPDATE users SET is_developer = TRUE WHERE id = $1;

-- name: SetUserRole :exec
UPDATE users SET role = $2 WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (token, user_id) VALUES ($1, $2);

-- name: GetUserByToken :one
SELECT users.* FROM users
JOIN sessions ON sessions.user_id = users.id
WHERE sessions.token = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;
