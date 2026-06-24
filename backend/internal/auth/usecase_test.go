package auth

import (
	"context"
	"errors"
	"testing"

	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// ---- fakes ----------------------------------------------------------------

type stored struct {
	user middleware.User
	hash string
}

type fakeRepo struct {
	byEmail  map[string]stored
	byName   map[string]middleware.User
	byID     map[string]middleware.User
	sessions map[string]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byEmail:  map[string]stored{},
		byName:   map[string]middleware.User{},
		byID:     map[string]middleware.User{},
		sessions: map[string]string{},
	}
}

func (r *fakeRepo) CreateUser(_ context.Context, id, username, email, hash string) (middleware.User, error) {
	u := middleware.User{ID: id, Username: username, Email: email, Role: middleware.RoleUser}
	r.byEmail[email] = stored{u, hash}
	r.byName[username] = u
	r.byID[id] = u
	return u, nil
}
func (r *fakeRepo) FindByEmail(_ context.Context, email string) (middleware.User, string, error) {
	s, ok := r.byEmail[email]
	if !ok {
		return middleware.User{}, "", ErrNotFound
	}
	return s.user, s.hash, nil
}
func (r *fakeRepo) FindByUsername(_ context.Context, username string) (middleware.User, error) {
	u, ok := r.byName[username]
	if !ok {
		return middleware.User{}, ErrNotFound
	}
	return u, nil
}
func (r *fakeRepo) FindByID(_ context.Context, id string) (middleware.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return middleware.User{}, ErrNotFound
	}
	return u, nil
}
func (r *fakeRepo) FindByToken(_ context.Context, token string) (middleware.User, error) {
	id, ok := r.sessions[token]
	if !ok {
		return middleware.User{}, ErrNotFound
	}
	return r.byID[id], nil
}
func (r *fakeRepo) CreateSession(_ context.Context, token, userID string) error {
	r.sessions[token] = userID
	return nil
}
func (r *fakeRepo) DeleteSession(_ context.Context, token string) error {
	delete(r.sessions, token)
	return nil
}

type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return "h:" + p, nil }
func (plainHasher) Compare(hash, p string) bool   { return hash == "h:"+p }

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Status
	}
	t.Fatalf("expected *apperr.Error, got %v", err)
	return 0
}

// ---- tests ------------------------------------------------------------

func TestRegister(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// seedUsername/seedEmail: if set, registered first so the case under
		// test collides with it.
		seedUsername, seedEmail   string
		username, email, password string
		wantErrStatus             int // 0 means success expected
		wantUsername, wantEmail   string
	}{
		{
			name:         "success normalizes email to lowercase",
			username:     "pixelsmith",
			email:        "Dev@Indie.GG",
			password:     "secret1",
			wantUsername: "pixelsmith",
			wantEmail:    "dev@indie.gg",
		},
		{
			name:          "duplicate email is rejected case-insensitively",
			seedUsername:  "a",
			seedEmail:     "x@y.z",
			username:      "b",
			email:         "X@Y.Z",
			password:      "secret1",
			wantErrStatus: 409,
		},
		{
			name:          "password under 6 chars is rejected",
			username:      "a",
			email:         "x@y.z",
			password:      "123",
			wantErrStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc := NewUseCase(newFakeRepo(), plainHasher{})
			ctx := context.Background()

			if tt.seedEmail != "" {
				if _, _, err := uc.Register(ctx, tt.seedUsername, tt.seedEmail, "secret1"); err != nil {
					t.Fatalf("seed register: %v", err)
				}
			}

			user, token, err := uc.Register(ctx, tt.username, tt.email, tt.password)

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user.Username != tt.wantUsername || user.Email != tt.wantEmail {
				t.Fatalf("unexpected user: %+v", user)
			}
			if token == "" {
				t.Fatal("expected a session token")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		password      string
		wantErrStatus int // 0 means success expected
	}{
		{name: "valid credentials succeed", password: "secret1"},
		{name: "wrong password is rejected", password: "wrong", wantErrStatus: 401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeRepo()
			uc := NewUseCase(repo, plainHasher{})
			ctx := context.Background()
			if _, _, err := uc.Register(ctx, "a", "x@y.z", "secret1"); err != nil {
				t.Fatal(err)
			}

			_, _, err := uc.Login(ctx, "x@y.z", tt.password)

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid login failed: %v", err)
			}
		})
	}
}
