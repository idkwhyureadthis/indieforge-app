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
	CreatedAt         time.Time
}

type repo struct{ q *sqlc.Queries }

// NewRepo builds the commerce repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapPayment(p sqlc.Payment) Payment {
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
		CreatedAt:         p.CreatedAt.Time,
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

func (r *repo) CreateSubscription(ctx context.Context, id, userID, gameID, developerID string, price int) error {
	_, err := r.q.CreateSubscription(ctx, sqlc.CreateSubscriptionParams{
		ID: id, UserID: userID, GameID: gameID, DeveloperID: developerID, Price: intconv.ToInt32(price),
	})
	return err
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
