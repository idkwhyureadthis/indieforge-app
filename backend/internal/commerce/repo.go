package commerce

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/middleware"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/intconv"
)

// ErrNotFound is returned when a payment, ownership, or user row is absent.
var ErrNotFound = errors.New("not found")

// Payment is the domain view of a payment row.
type Payment struct {
	ID                string
	YkID              string
	UserID            string
	GameID            string
	Kind              string
	Amount            int
	CommissionPercent int
	CommissionAmount  int
	Status            string
	FriendUsername    string
	PlanID            string // empty for regular payments; set for plan subscription payments
	SubID             string // set for renewal payments; links back to the subscription being renewed
	PaymentMethodID   string // YooKassa saved payment method ID (set on succeeded subscription payments)
	CreatedAt         time.Time
}

// VerifyResult is the result of a subscription status check.
type VerifyResult struct {
	Subscribed bool
	ExpiresAt  *time.Time
}

// Subscription is the domain view of a subscription row.
type Subscription struct {
	ID              string
	UserID          string
	GameID          string
	DeveloperID     string
	Price           int
	Active          bool
	PaymentMethodID string
	ExpiresAt       *time.Time
	StartedAt       time.Time
}

type repo struct{ q *sqlc.Queries }

// NewRepo builds the commerce repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapPayment(p sqlc.Payment) Payment {
	planID := ""
	if p.PlanID != nil {
		planID = *p.PlanID
	}
	subID := ""
	if p.SubID != nil {
		subID = *p.SubID
	}
	return Payment{
		ID:                p.ID,
		YkID:              p.YkID,
		UserID:            p.UserID,
		GameID:            p.GameID,
		Kind:              p.Kind,
		Amount:            int(p.Amount),
		CommissionPercent: int(p.CommissionPercent),
		CommissionAmount:  int(p.CommissionAmount),
		Status:            p.Status,
		FriendUsername:    p.FriendUsername,
		PlanID:            planID,
		SubID:             subID,
		PaymentMethodID:   p.PaymentMethodID,
		CreatedAt:         p.CreatedAt.Time,
	}
}

func mapSubscription(s sqlc.Subscription) Subscription {
	return Subscription{
		ID:              s.ID,
		UserID:          s.UserID,
		GameID:          s.GameID,
		DeveloperID:     s.DeveloperID,
		Price:           int(s.Price),
		Active:          s.Active,
		PaymentMethodID: s.PaymentMethodID,
		ExpiresAt:       s.ExpiresAt,
		StartedAt:       s.StartedAt.Time,
	}
}

func (r *repo) CreateOwnership(ctx context.Context, id, userID, gameID, otype string, price int, giftedBy string) error {
	_, err := r.q.CreateOwnership(ctx, sqlc.CreateOwnershipParams{
		ID: id, UserID: userID, GameID: gameID, Type: otype, Price: intconv.ToInt32(price), GiftedBy: giftedBy,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // ON CONFLICT DO NOTHING — already owned
	}
	return err
}

func (r *repo) HasOwnership(ctx context.Context, userID, gameID string) (bool, error) {
	_, err := r.q.GetOwnership(ctx, sqlc.GetOwnershipParams{UserID: userID, GameID: gameID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *repo) OwnedGameIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.q.ListOwnershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, o := range rows {
		ids = append(ids, o.GameID)
	}
	return ids, nil
}

func (r *repo) CreateSubscription(ctx context.Context, id, userID, gameID, developerID string, price int) (Subscription, error) {
	s, err := r.q.CreateSubscription(ctx, sqlc.CreateSubscriptionParams{
		ID: id, UserID: userID, GameID: gameID, DeveloperID: developerID, Price: intconv.ToInt32(price),
	})
	if err != nil {
		return Subscription{}, err
	}
	return mapSubscription(s), nil
}

func (r *repo) GetSubscriptionByID(ctx context.Context, id string) (Subscription, error) {
	s, err := r.q.GetSubscriptionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, err
	}
	return mapSubscription(s), nil
}

func (r *repo) SetSubscriptionRenewalInfo(ctx context.Context, subID string, expiresAt time.Time, paymentMethodID string) error {
	return r.q.SetSubscriptionRenewalInfo(ctx, sqlc.SetSubscriptionRenewalInfoParams{
		ID:              subID,
		ExpiresAt:       &expiresAt,
		PaymentMethodID: paymentMethodID,
	})
}

func (r *repo) ExtendSubscription(ctx context.Context, subID string, expiresAt time.Time) error {
	return r.q.ExtendSubscription(ctx, sqlc.ExtendSubscriptionParams{
		ID:        subID,
		ExpiresAt: &expiresAt,
	})
}

func (r *repo) DeactivateSubscription(ctx context.Context, subID string) error {
	return r.q.DeactivateSubscription(ctx, subID)
}

func (r *repo) ListExpiringSubscriptions(ctx context.Context, before time.Time) ([]Subscription, error) {
	t := before
	rows, err := r.q.ListExpiringSubscriptions(ctx, &t)
	if err != nil {
		return nil, err
	}
	out := make([]Subscription, len(rows))
	for i, s := range rows {
		out[i] = mapSubscription(s)
	}
	return out, nil
}

func (r *repo) SetPaymentSubID(ctx context.Context, paymentID, subID string) error {
	return r.q.SetPaymentSubID(ctx, sqlc.SetPaymentSubIDParams{ID: paymentID, SubID: &subID})
}

func (r *repo) SetPaymentMethodID(ctx context.Context, paymentID, methodID string) error {
	return r.q.SetPaymentMethodID(ctx, sqlc.SetPaymentMethodIDParams{ID: paymentID, PaymentMethodID: methodID})
}

func (r *repo) GetUserSubscriptionStatus(ctx context.Context, userID, gameKey string) (VerifyResult, error) {
	row, err := r.q.GetUserSubscriptionStatus(ctx, sqlc.GetUserSubscriptionStatusParams{
		UserID: userID,
		ID:     gameKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return VerifyResult{Subscribed: false}, nil
	}
	if err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{Subscribed: row.Active, ExpiresAt: row.ExpiresAt}, nil
}

func (r *repo) GetGameIDByKey(ctx context.Context, key string) (string, error) {
	id, err := r.q.GetGameIDBySlugOrID(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

func (r *repo) CreateLaunchToken(ctx context.Context, tokenHash, userID, gameID string) error {
	return r.q.CreateLaunchToken(ctx, sqlc.CreateLaunchTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
		GameID:    gameID,
	})
}

func (r *repo) ListSubscriptions(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := r.q.ListSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Subscription, len(rows))
	for i, s := range rows {
		out[i] = mapSubscription(s)
	}
	return out, nil
}

func (r *repo) HasActiveSubscription(ctx context.Context, userID, gameID string) (bool, error) {
	_, err := r.q.GetActiveSubscription(ctx, sqlc.GetActiveSubscriptionParams{UserID: userID, GameID: gameID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *repo) SubscribedGameIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.q.ListSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, s := range rows {
		ids = append(ids, s.GameID)
	}
	return ids, nil
}

func (r *repo) CreatePayment(ctx context.Context, p Payment) (Payment, error) {
	out, err := r.q.CreatePayment(ctx, sqlc.CreatePaymentParams{
		ID:                p.ID,
		YkID:              p.YkID,
		UserID:            p.UserID,
		GameID:            p.GameID,
		Kind:              p.Kind,
		Amount:            intconv.ToInt32(p.Amount),
		CommissionPercent: intconv.ToInt32(p.CommissionPercent),
		CommissionAmount:  intconv.ToInt32(p.CommissionAmount),
		Status:            p.Status,
		FriendUsername:    p.FriendUsername,
	})
	if err != nil {
		return Payment{}, err
	}
	return mapPayment(out), nil
}

func (r *repo) GetPaymentByID(ctx context.Context, id string) (Payment, error) {
	p, err := r.q.GetPaymentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, err
	}
	return mapPayment(p), nil
}

func (r *repo) GetPaymentByYkID(ctx context.Context, ykID string) (Payment, error) {
	p, err := r.q.GetPaymentByYkID(ctx, ykID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, err
	}
	return mapPayment(p), nil
}

func (r *repo) SetPaymentYkID(ctx context.Context, id, ykID string) error {
	return r.q.SetPaymentYkID(ctx, sqlc.SetPaymentYkIDParams{ID: id, YkID: ykID})
}

func (r *repo) UpdatePaymentStatus(ctx context.Context, id, status string) error {
	return r.q.UpdatePaymentStatus(ctx, sqlc.UpdatePaymentStatusParams{ID: id, Status: status})
}

func (r *repo) DeleteOwnership(ctx context.Context, userID, gameID string) error {
	return r.q.DeleteOwnership(ctx, sqlc.DeleteOwnershipParams{UserID: userID, GameID: gameID})
}

// SubscriptionPlan is the domain view of a developer plan (used in commerce for payment processing).
type SubscriptionPlan struct {
	ID          string
	DeveloperID string
	Price       int
}

func (r *repo) GetSubscriptionPlan(ctx context.Context, planID string) (SubscriptionPlan, error) {
	p, err := r.q.GetPlanByID(ctx, planID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubscriptionPlan{}, ErrNotFound
	}
	if err != nil {
		return SubscriptionPlan{}, err
	}
	return SubscriptionPlan{ID: p.ID, DeveloperID: p.DeveloperID, Price: int(p.Price)}, nil
}

func (r *repo) ListPlanGameIDs(ctx context.Context, planID string) ([]string, error) {
	rows, err := r.q.ListPlanGameIDs(ctx, planID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repo) SetPaymentPlanID(ctx context.Context, paymentID, planID string) error {
	return r.q.SetPaymentPlanID(ctx, sqlc.SetPaymentPlanIDParams{ID: paymentID, PlanID: &planID})
}

func (r *repo) UserByUsername(ctx context.Context, username string) (middleware.User, error) {
	u, err := r.q.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.User{}, ErrNotFound
	}
	if err != nil {
		return middleware.User{}, err
	}
	return middleware.User{ID: u.ID, Username: u.Username, Email: u.Email, Role: middleware.Role(u.Role), IsDeveloper: u.IsDeveloper, CreatedAt: u.CreatedAt.Time}, nil
}

func (r *repo) UsernameByID(ctx context.Context, id string) (string, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return "", err
	}
	return u.Username, nil
}
