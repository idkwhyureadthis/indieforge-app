package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/middleware"
	"indieforge/internal/platform/db/sqlc"
)

// ErrNotFound is returned by the repo when a row is absent; usecases match on it.
var ErrNotFound = errors.New("not found")

type repo struct{ q *sqlc.Queries }

// NewRepo builds the auth repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapUser(u sqlc.User) middleware.User {
	return middleware.User{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Role:        middleware.Role(u.Role),
		IsDeveloper: u.IsDeveloper,
		CreatedAt:   u.CreatedAt.Time,
	}
}

func (r *repo) CreateUser(ctx context.Context, id, username, email, hash string) (middleware.User, error) {
	u, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{ID: id, Username: username, Email: email, PasswordHash: hash})
	if err != nil {
		return middleware.User{}, err
	}
	return mapUser(u), nil
}

func (r *repo) FindByEmail(ctx context.Context, email string) (middleware.User, string, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.User{}, "", ErrNotFound
	}
	if err != nil {
		return middleware.User{}, "", err
	}
	return mapUser(u), u.PasswordHash, nil
}

func (r *repo) FindByUsername(ctx context.Context, username string) (middleware.User, error) {
	u, err := r.q.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.User{}, ErrNotFound
	}
	if err != nil {
		return middleware.User{}, err
	}
	return mapUser(u), nil
}

func (r *repo) FindByID(ctx context.Context, id string) (middleware.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.User{}, ErrNotFound
	}
	if err != nil {
		return middleware.User{}, err
	}
	return mapUser(u), nil
}

func (r *repo) FindByToken(ctx context.Context, token string) (middleware.User, error) {
	u, err := r.q.GetUserByToken(ctx, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.User{}, ErrNotFound
	}
	if err != nil {
		return middleware.User{}, err
	}
	return mapUser(u), nil
}

func (r *repo) CreateSession(ctx context.Context, token, userID string) error {
	return r.q.CreateSession(ctx, sqlc.CreateSessionParams{Token: token, UserID: userID})
}

func (r *repo) DeleteSession(ctx context.Context, token string) error {
	return r.q.DeleteSession(ctx, token)
}
