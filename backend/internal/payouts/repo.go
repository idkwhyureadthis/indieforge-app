package payouts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/intconv"
)

var ErrNotFound = errors.New("not found")

type Payout struct {
	ID          string
	DeveloperID string
	Amount      int
	Status      string
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PayoutWithDev struct {
	Payout
	DeveloperUsername string
}

type repo struct{ q *sqlc.Queries }

func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapPayout(p sqlc.Payout) Payout {
	return Payout{
		ID:          p.ID,
		DeveloperID: p.DeveloperID,
		Amount:      int(p.Amount),
		Status:      p.Status,
		Note:        p.Note,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}

func (r *repo) GetBalance(ctx context.Context, developerID string) (earned int, requested int, err error) {
	e, err := r.q.GetDeveloperEarnings(ctx, developerID)
	if err != nil {
		return 0, 0, err
	}
	t, err := r.q.GetDeveloperPayoutsTotal(ctx, developerID)
	if err != nil {
		return 0, 0, err
	}
	return int(e), int(t), nil
}

func (r *repo) CreatePayout(ctx context.Context, id, developerID string, amount int) (Payout, error) {
	p, err := r.q.CreatePayout(ctx, sqlc.CreatePayoutParams{
		ID: id, DeveloperID: developerID, Amount: intconv.ToInt32(amount),
	})
	if err != nil {
		return Payout{}, err
	}
	return mapPayout(p), nil
}

func (r *repo) GetPayoutByID(ctx context.Context, id string) (Payout, error) {
	p, err := r.q.GetPayoutByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payout{}, ErrNotFound
	}
	if err != nil {
		return Payout{}, err
	}
	return mapPayout(p), nil
}

func (r *repo) ListByDeveloper(ctx context.Context, developerID string) ([]Payout, error) {
	rows, err := r.q.ListPayoutsByDeveloper(ctx, developerID)
	if err != nil {
		return nil, err
	}
	out := make([]Payout, len(rows))
	for i, p := range rows {
		out[i] = mapPayout(p)
	}
	return out, nil
}

func (r *repo) ListAll(ctx context.Context) ([]PayoutWithDev, error) {
	rows, err := r.q.ListAllPayouts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PayoutWithDev, len(rows))
	for i, p := range rows {
		out[i] = PayoutWithDev{
			Payout:            mapPayout(sqlc.Payout{ID: p.ID, DeveloperID: p.DeveloperID, Amount: p.Amount, Status: p.Status, Note: p.Note, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}),
			DeveloperUsername: p.DeveloperUsername,
		}
	}
	return out, nil
}

func (r *repo) UpdateStatus(ctx context.Context, id, status, note string) (Payout, error) {
	p, err := r.q.UpdatePayoutStatus(ctx, sqlc.UpdatePayoutStatusParams{ID: id, Status: status, Note: note})
	if errors.Is(err, pgx.ErrNoRows) {
		return Payout{}, ErrNotFound
	}
	if err != nil {
		return Payout{}, err
	}
	return mapPayout(p), nil
}
