package commerce

import (
	"context"
	"errors"
	"testing"
	"time"

	"indieforge/internal/dto"
	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/internal/platform/yookassa"
	"indieforge/pkg/apperr"
)

// ---- fakes --------------------------------------------------------------

type fakeRepo struct {
	owns      map[string]bool // userID|gameID
	subs      map[string]bool
	payments  map[string]Payment
	ykIndex   map[string]string // ykID -> paymentID
	usersByNm map[string]middleware.User
	usersByID map[string]string // id -> username
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		owns:      map[string]bool{},
		subs:      map[string]bool{},
		payments:  map[string]Payment{},
		ykIndex:   map[string]string{},
		usersByNm: map[string]middleware.User{},
		usersByID: map[string]string{},
	}
}

func key(a, b string) string { return a + "|" + b }

func (r *fakeRepo) CreateOwnership(_ context.Context, _, userID, gameID, _ string, _ int, _ string) error {
	r.owns[key(userID, gameID)] = true
	return nil
}
func (r *fakeRepo) HasOwnership(_ context.Context, userID, gameID string) (bool, error) {
	return r.owns[key(userID, gameID)], nil
}
func (r *fakeRepo) OwnedGameIDs(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (r *fakeRepo) CreateSubscription(_ context.Context, id, userID, gameID, _ string, _ int) (Subscription, error) {
	r.subs[key(userID, gameID)] = true
	return Subscription{ID: id, UserID: userID, GameID: gameID}, nil
}
func (r *fakeRepo) GetSubscriptionByID(_ context.Context, _ string) (Subscription, error) {
	return Subscription{}, ErrNotFound
}
func (r *fakeRepo) SetSubscriptionRenewalInfo(_ context.Context, _ string, _ time.Time, _ string) error {
	return nil
}
func (r *fakeRepo) ExtendSubscription(_ context.Context, _ string, _ time.Time) error { return nil }
func (r *fakeRepo) DeactivateSubscription(_ context.Context, _ string) error           { return nil }
func (r *fakeRepo) ListExpiringSubscriptions(_ context.Context, _ time.Time) ([]Subscription, error) {
	return nil, nil
}
func (r *fakeRepo) SetPaymentSubID(_ context.Context, _, _ string) error    { return nil }
func (r *fakeRepo) SetPaymentMethodID(_ context.Context, _, _ string) error { return nil }
func (r *fakeRepo) HasActiveSubscription(_ context.Context, userID, gameID string) (bool, error) {
	return r.subs[key(userID, gameID)], nil
}
func (r *fakeRepo) SubscribedGameIDs(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (r *fakeRepo) ListSubscriptions(_ context.Context, _ string) ([]Subscription, error) {
	return nil, nil
}
func (r *fakeRepo) GetUserSubscriptionStatus(_ context.Context, _, _ string) (VerifyResult, error) {
	return VerifyResult{}, nil
}
func (r *fakeRepo) GetGameIDByKey(_ context.Context, key string) (string, error) { return key, nil }
func (r *fakeRepo) CreateLaunchToken(_ context.Context, _, _, _ string) error    { return nil }
func (r *fakeRepo) CreatePayment(_ context.Context, p Payment) (Payment, error) {
	r.payments[p.ID] = p
	return p, nil
}
func (r *fakeRepo) GetPaymentByID(_ context.Context, id string) (Payment, error) {
	p, ok := r.payments[id]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return p, nil
}
func (r *fakeRepo) GetPaymentByYkID(_ context.Context, ykID string) (Payment, error) {
	id, ok := r.ykIndex[ykID]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return r.payments[id], nil
}
func (r *fakeRepo) SetPaymentYkID(_ context.Context, id, ykID string) error {
	p := r.payments[id]
	p.YkID = ykID
	r.payments[id] = p
	r.ykIndex[ykID] = id
	return nil
}
func (r *fakeRepo) UpdatePaymentStatus(_ context.Context, id, status string) error {
	p := r.payments[id]
	p.Status = status
	r.payments[id] = p
	return nil
}
func (r *fakeRepo) UserByUsername(_ context.Context, username string) (middleware.User, error) {
	u, ok := r.usersByNm[username]
	if !ok {
		return middleware.User{}, ErrNotFound
	}
	return u, nil
}
func (r *fakeRepo) UsernameByID(_ context.Context, id string) (string, error) {
	return r.usersByID[id], nil
}
func (r *fakeRepo) DeleteOwnership(_ context.Context, userID, gameID string) error {
	delete(r.owns, key(userID, gameID))
	return nil
}
func (r *fakeRepo) GetSubscriptionPlan(_ context.Context, _ string) (SubscriptionPlan, error) {
	return SubscriptionPlan{}, ErrNotFound
}
func (r *fakeRepo) ListPlanGameIDs(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (r *fakeRepo) SetPaymentPlanID(_ context.Context, _, _ string) error          { return nil }

type fakeGames struct{ games map[string]games.Game }

func (f *fakeGames) GameByKey(_ context.Context, key string) (games.Game, error) {
	g, ok := f.games[key]
	if !ok {
		return games.Game{}, games.ErrNotFound
	}
	return g, nil
}
func (f *fakeGames) Serialize(_ context.Context, g games.Game, _ string) (dto.GameDTO, error) {
	return dto.GameDTO{ID: g.ID, Slug: g.Slug, Title: g.Title}, nil
}
func (f *fakeGames) RecordEvent(_ context.Context, _, _ string) error { return nil }

type fakeYK struct{ status string }

func (y *fakeYK) Configured() bool { return true }
func (y *fakeYK) CreatePayment(_ context.Context, _ yookassa.CreateParams) (yookassa.Payment, error) {
	return yookassa.Payment{ID: "yk_1", Status: "pending", ConfirmationURL: "https://yoo/confirm"}, nil
}
func (y *fakeYK) GetPayment(_ context.Context, id string) (yookassa.Payment, error) {
	return yookassa.Payment{ID: id, Status: y.status}, nil
}
func (y *fakeYK) CreateRecurrentPayment(_ context.Context, _ yookassa.RecurrentParams) (yookassa.Payment, error) {
	return yookassa.Payment{ID: "yk_renew", Status: "pending"}, nil
}
func (y *fakeYK) RefundPayment(_ context.Context, _ string, _ int) error { return nil }

type fakeSettings struct{ commission int }

func (s fakeSettings) Commission(_ context.Context) (int, error) { return s.commission, nil }

func setup() (*UseCase, *fakeRepo, *fakeGames) {
	repo := newFakeRepo()
	fg := &fakeGames{games: map[string]games.Game{}}
	uc := NewUseCase(repo, fg, &fakeYK{status: "succeeded"}, fakeSettings{commission: 10}, "http://localhost:5173")
	return uc, repo, fg
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

func TestClaimFree(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		game          games.Game
		alreadyOwns   bool
		wantErrStatus int // 0 means success expected
	}{
		{
			name: "free game grants ownership",
			game: games.Game{ID: "g1", PricingModel: "free"},
		},
		{
			name:          "already owned is a conflict",
			game:          games.Game{ID: "g1", PricingModel: "free"},
			alreadyOwns:   true,
			wantErrStatus: 409,
		},
		{
			name:          "paid game is rejected",
			game:          games.Game{ID: "g1", PricingModel: "paid", Price: 499},
			wantErrStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc, repo, fg := setup()
			fg.games["g1"] = tt.game
			user := middleware.User{ID: "u1", Username: "u1"}
			if tt.alreadyOwns {
				repo.owns[key("u1", "g1")] = true
			}

			_, err := uc.ClaimFree(context.Background(), user, "g1")

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("claim free failed: %v", err)
			}
			if !repo.owns[key("u1", "g1")] {
				t.Fatal("expected ownership granted")
			}
		})
	}
}

func TestCreatePayment_FriendPack(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		buyerOwns      bool
		friendUsername string
		friendExists   bool
		friendOwns     bool
		wantErrStatus  int // 0 means success expected
		wantAmount     int
		wantCommission int
	}{
		{
			name:           "buyer who doesn't own the game is forbidden",
			friendUsername: "friend",
			friendExists:   true,
			wantErrStatus:  403,
		},
		{
			name:           "unknown friend is not found",
			buyerOwns:      true,
			friendUsername: "nobody",
			wantErrStatus:  404,
		},
		{
			name:           "happy path applies the friend-pack discount and commission",
			buyerOwns:      true,
			friendUsername: "friend",
			friendExists:   true,
			wantAmount:     300, // 500 * (1 - 40%)
			wantCommission: 30,  // 10% of 300
		},
		{
			name:           "friend who already owns the game is a conflict",
			buyerOwns:      true,
			friendUsername: "friend",
			friendExists:   true,
			friendOwns:     true,
			wantErrStatus:  409,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc, repo, fg := setup()
			fg.games["g1"] = games.Game{ID: "g1", PricingModel: "paid", Price: 500, FriendPackDiscount: 40}
			buyer := middleware.User{ID: "u1", Username: "buyer"}
			if tt.buyerOwns {
				repo.owns[key("u1", "g1")] = true
			}
			if tt.friendExists {
				repo.usersByNm["friend"] = middleware.User{ID: "u2", Username: "friend"}
				if tt.friendOwns {
					repo.owns[key("u2", "g1")] = true
				}
			}

			pay, url, err := uc.CreatePayment(context.Background(), buyer, "g1", "friend-pack", tt.friendUsername, "")

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("friend-pack failed: %v", err)
			}
			if pay.Amount != tt.wantAmount {
				t.Fatalf("expected amount %d, got %d", tt.wantAmount, pay.Amount)
			}
			if pay.CommissionAmount != tt.wantCommission {
				t.Fatalf("expected commission %d, got %d", tt.wantCommission, pay.CommissionAmount)
			}
			if url == "" {
				t.Fatal("expected a confirmation url")
			}
		})
	}
}

// TestWebhook_GrantsAndIsIdempotent intentionally shares state across
// subtests (steps), since idempotency is a property of repeated delivery
// against the same payment — not something independent table rows can
// express on their own.
//
//nolint:tparallel // subtests deliberately run in sequence, see comment below
func TestWebhook_GrantsAndIsIdempotent(t *testing.T) {
	t.Parallel()
	uc, repo, fg := setup()
	fg.games["g1"] = games.Game{ID: "g1", PricingModel: "paid", Price: 499}
	buyer := middleware.User{ID: "u1", Username: "buyer"}

	pay, _, err := uc.CreatePayment(context.Background(), buyer, "g1", "purchase", "", "")
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	body := []byte(`{"event":"payment.succeeded","object":{"id":"` + pay.YkID + `","status":"succeeded"}}`)

	steps := []struct {
		name string
	}{
		{name: "first delivery grants ownership"},
		{name: "second delivery is a no-op"},
	}
	//nolint:paralleltest // these steps deliberately share state in sequence;
	// parallelizing them would race the two deliveries against the same fake repo.
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			if err := uc.HandleWebhook(context.Background(), body); err != nil {
				t.Fatalf("webhook: %v", err)
			}
			if !repo.owns[key("u1", "g1")] {
				t.Fatal("expected ownership after webhook")
			}
			if repo.payments[pay.ID].Status != "succeeded" {
				t.Fatal("expected payment marked succeeded")
			}
		})
	}
}

func TestPerks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		userID        string
		isSubscribed  bool
		wantErrStatus int // 0 means success expected
		wantLink      string
	}{
		{name: "non-subscriber is rejected", userID: "u1", wantErrStatus: 403},
		{name: "subscriber gets the chat link", userID: "u1", isSubscribed: true, wantLink: "https://discord.gg/forge"},
		{name: "author gets the link without subscribing", userID: "dev", wantLink: "https://discord.gg/forge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc, repo, fg := setup()
			fg.games["g1"] = games.Game{
				ID: "g1", DeveloperID: "dev",
				Subscription: games.Subscription{Enabled: true, ChatLink: "https://discord.gg/forge"},
			}
			if tt.isSubscribed {
				repo.subs[key(tt.userID, "g1")] = true
			}

			link, err := uc.Perks(context.Background(), middleware.User{ID: tt.userID}, "g1")

			if tt.wantErrStatus != 0 {
				if got := statusOf(t, err); got != tt.wantErrStatus {
					t.Fatalf("expected status %d, got %d", tt.wantErrStatus, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if link != tt.wantLink {
				t.Fatalf("expected link %q, got %q", tt.wantLink, link)
			}
		})
	}
}
