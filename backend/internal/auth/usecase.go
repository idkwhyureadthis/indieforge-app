package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
)

// Repo is the persistence port the auth usecase needs.
type Repo interface {
	CreateUser(ctx context.Context, id, username, email, hash string) (middleware.User, error)
	FindByEmail(ctx context.Context, email string) (middleware.User, string, error)
	FindByUsername(ctx context.Context, username string) (middleware.User, error)
	FindByID(ctx context.Context, id string) (middleware.User, error)
	FindByToken(ctx context.Context, token string) (middleware.User, error)
	CreateSession(ctx context.Context, token, userID string) error
	DeleteSession(ctx context.Context, token string) error
}

// Hasher is the password-hashing port.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

// UseCase implements the auth business rules.
type UseCase struct {
	repo   Repo
	hasher Hasher
}

// NewUseCase wires the auth usecase to its ports.
func NewUseCase(repo Repo, hasher Hasher) *UseCase {
	return &UseCase{repo: repo, hasher: hasher}
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Register creates a new account and returns the user plus a session token.
func (uc *UseCase) Register(ctx context.Context, username, email, password string) (middleware.User, string, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	if username == "" || email == "" || password == "" {
		return middleware.User{}, "", apperr.BadRequest("Fill in name, email and password")
	}
	if len(password) < 6 {
		return middleware.User{}, "", apperr.BadRequest("Password must be at least 6 characters")
	}

	if _, _, err := uc.repo.FindByEmail(ctx, email); err == nil {
		return middleware.User{}, "", apperr.Conflict("An account with this email already exists")
	} else if !errors.Is(err, ErrNotFound) {
		return middleware.User{}, "", err
	}
	if _, err := uc.repo.FindByUsername(ctx, username); err == nil {
		return middleware.User{}, "", apperr.Conflict("That username is taken")
	} else if !errors.Is(err, ErrNotFound) {
		return middleware.User{}, "", err
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return middleware.User{}, "", apperr.Internal("Could not hash password")
	}
	user, err := uc.repo.CreateUser(ctx, idgen.New("usr"), username, email, hash)
	if err != nil {
		return middleware.User{}, "", err
	}
	token := newToken()
	if err := uc.repo.CreateSession(ctx, token, user.ID); err != nil {
		return middleware.User{}, "", err
	}
	return user, token, nil
}

// Login verifies credentials and returns the user plus a new session token.
func (uc *UseCase) Login(ctx context.Context, email, password string) (middleware.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, hash, err := uc.repo.FindByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) || (err == nil && !uc.hasher.Compare(hash, password)) {
		return middleware.User{}, "", apperr.Unauthorized("Invalid email or password")
	}
	if err != nil {
		return middleware.User{}, "", err
	}
	token := newToken()
	if err := uc.repo.CreateSession(ctx, token, user.ID); err != nil {
		return middleware.User{}, "", err
	}
	return user, token, nil
}

// Logout deletes the session behind the given token.
func (uc *UseCase) Logout(ctx context.Context, token string) error {
	return uc.repo.DeleteSession(ctx, token)
}
