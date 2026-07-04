package games

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"indieforge/internal/middleware"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/apperr"
)

// ---- stub ports (only what Create/DownloadURL/Home touch) ---------------

type stubRepo struct{}

func (stubRepo) Create(context.Context, sqlc.CreateGameParams) (Game, error) { return Game{}, nil }
func (stubRepo) GetByID(context.Context, string) (Game, error)               { return Game{}, ErrNotFound }
func (stubRepo) GetBySlug(context.Context, string) (Game, error)             { return Game{}, ErrNotFound }
func (stubRepo) ListPublished(context.Context) ([]Game, error)               { return nil, nil }
func (stubRepo) ListByDeveloper(context.Context, string) ([]Game, error)     { return nil, nil }
func (stubRepo) ListNewest(context.Context, int) ([]Game, error)             { return nil, nil }
func (stubRepo) ListTrending(context.Context, int) ([]Game, error)           { return nil, nil }
func (stubRepo) ListPopular(context.Context, int) ([]Game, error)            { return nil, nil }
func (stubRepo) SlugExists(context.Context, string) (bool, error)            { return false, nil }
func (stubRepo) SetStatus(context.Context, string, string) error             { return nil }
func (stubRepo) InsertEvent(context.Context, string, string, string) error   { return nil }
func (stubRepo) RecomputeTrending(context.Context) error                     { return nil }
func (stubRepo) OwnerCounts(context.Context) (map[string]int, error)         { return map[string]int{}, nil }
func (stubRepo) SubscriberCounts(context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}
func (stubRepo) CountOwners(context.Context, string) (int, error)                   { return 0, nil }
func (stubRepo) CountSubscribers(context.Context, string) (int, error)              { return 0, nil }
func (stubRepo) HasOwnership(context.Context, string, string) (bool, error)         { return false, nil }
func (stubRepo) HasSubscription(context.Context, string, string) (bool, error)      { return false, nil }
func (stubRepo) OwnedGameIDs(context.Context, string) (map[string]bool, error)      { return nil, nil }
func (stubRepo) SubscribedGameIDs(context.Context, string) (map[string]bool, error) { return nil, nil }
func (stubRepo) MarkDeveloper(context.Context, string) error                        { return nil }

type stubStorage struct{ putCalled bool }

func (s *stubStorage) PutPublic(context.Context, string, string, []byte) (string, error) {
	s.putCalled = true
	return "url", nil
}
func (s *stubStorage) PutPrivate(context.Context, string, string, []byte) error {
	s.putCalled = true
	return nil
}
func (s *stubStorage) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "url", nil
}
func (s *stubStorage) ExtractZipToPrefix(context.Context, string, []byte) (string, error) {
	return "url/index.html", nil
}

type scanner struct {
	clean bool
	sig   string
}

func (s scanner) Scan(_ context.Context, r io.Reader) (bool, string, error) {
	_, _ = io.Copy(io.Discard, r)
	return s.clean, s.sig, nil
}

// downloadRepo stubs just enough of Repo to exercise DownloadURL's access checks.
type downloadRepo struct {
	stubRepo
	game     Game
	ownerIDs map[string]bool
}

func (r downloadRepo) GetBySlug(context.Context, string) (Game, error) { return r.game, nil }
func (r downloadRepo) HasOwnership(_ context.Context, userID, _ string) (bool, error) {
	return r.ownerIDs[userID], nil
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Status
	}
	t.Fatalf("expected *apperr.Error, got %v", err)
	return 0
}

// ---- tests --------------------------------------------------------------

func TestCreate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		input           NewGame
		scannerClean    bool
		scannerSig      string
		wantErrStatus   int
		wantStoreCalled bool
	}{
		{
			name:          "no build provided is rejected",
			input:         NewGame{Title: "X"},
			scannerClean:  true,
			wantErrStatus: 400,
		},
		{
			name: "antivirus rejects an infected upload before storing anything",
			input: NewGame{
				Title:            "X",
				HasDownloadBuild: true,
				DownloadFile:     &Upload{Filename: "game.zip", Data: []byte("malware")},
			},
			scannerClean:  false,
			scannerSig:    "Eicar-Test-Signature",
			wantErrStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &stubStorage{}
			uc := NewUseCase(stubRepo{}, store, scanner{clean: tt.scannerClean, sig: tt.scannerSig})

			_, err := uc.Create(context.Background(), middleware.User{ID: "u1"}, tt.input)

			if got := statusOf(t, err); got != tt.wantErrStatus {
				t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
			}
			if store.putCalled != tt.wantStoreCalled {
				t.Fatalf("store.putCalled = %v, want %v", store.putCalled, tt.wantStoreCalled)
			}
		})
	}
}

func TestHome_SectionsAreNeverNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                            string
		trendingEnabled, popularEnabled bool
	}{
		{name: "both disabled (default)", trendingEnabled: false, popularEnabled: false},
		{name: "only trending enabled", trendingEnabled: true, popularEnabled: false},
		{name: "only popular enabled", trendingEnabled: false, popularEnabled: true},
		{name: "both enabled", trendingEnabled: true, popularEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc := NewUseCase(stubRepo{}, &stubStorage{}, scanner{clean: true})

			out, err := uc.Home(context.Background(), "", tt.trendingEnabled, tt.popularEnabled, 12)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Must be non-nil empty slices (marshal to `[]`), not nil (marshals
			// to `null`) — the frontend calls array methods on these unconditionally.
			if out.Trending == nil || out.Popular == nil || out.Newest == nil || out.DemoDay == nil {
				t.Fatal("all sections must serialize as [], never null")
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple title", in: "Emberfall", want: "emberfall"},
		{name: "punctuation is stripped", in: "Neon Drift!", want: "neon-drift"},
		{name: "extra whitespace collapses to one dash", in: "  Last   Hearth  ", want: "last-hearth"},
		{name: "empty input falls back to game", in: "", want: "game"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := slugify(tt.in); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDemoDayActive(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		demo DemoDay
		want bool
	}{
		{name: "disabled is always inactive", demo: DemoDay{Enabled: false}, want: false},
		{name: "within the start/end window is active", demo: DemoDay{Enabled: true, StartsAt: &past, EndsAt: &future}, want: true},
		{name: "starting in the future is inactive", demo: DemoDay{Enabled: true, StartsAt: &future}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.demo.Active(); got != tt.want {
				t.Errorf("Active() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownloadURL(t *testing.T) {
	t.Parallel()
	game := Game{
		ID: "g1", Slug: "g1", DeveloperID: "dev1",
		PricingModel: "paid", Price: 499,
		HasDownloadBuild: true, DownloadObjectKey: "downloads/g1/game.zip",
	}

	tests := []struct {
		name          string
		viewerID      string
		ownerIDs      map[string]bool
		wantErrStatus int // 0 means success expected
	}{
		{
			name:     "the author can download their own unbought game",
			viewerID: "dev1",
			ownerIDs: map[string]bool{},
		},
		{
			name:     "an owner can download",
			viewerID: "buyer",
			ownerIDs: map[string]bool{"buyer": true},
		},
		{
			name:          "a stranger who neither owns nor authored it is rejected",
			viewerID:      "stranger",
			ownerIDs:      map[string]bool{},
			wantErrStatus: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := downloadRepo{game: game, ownerIDs: tt.ownerIDs}
			uc := NewUseCase(repo, &stubStorage{}, scanner{clean: true})

			_, err := uc.DownloadURL(context.Background(), "g1", tt.viewerID)

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCanPlayFree(t *testing.T) {
	t.Parallel()
	paid := Game{PricingModel: "paid", Price: 499}
	free := Game{PricingModel: "free"}

	tests := []struct {
		name   string
		game   Game
		viewer ViewerContext
		want   bool
	}{
		{name: "unowned paid game is not playable free", game: paid, viewer: ViewerContext{Owned: false}, want: false},
		{name: "owned paid game is playable", game: paid, viewer: ViewerContext{Owned: true}, want: true},
		{name: "subscribed paid game is playable", game: paid, viewer: ViewerContext{Subscribed: true}, want: true},
		{name: "free game is always playable", game: free, viewer: ViewerContext{}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := toDTO(tt.game, Counts{}, tt.viewer).CanPlayFree; got != tt.want {
				t.Errorf("CanPlayFree = %v, want %v", got, tt.want)
			}
		})
	}
}
