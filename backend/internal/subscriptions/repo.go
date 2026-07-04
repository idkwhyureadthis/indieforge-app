package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/intconv"
)

// ErrNotFound is returned when no plan row exists.
var ErrNotFound = errors.New("plan not found")

// Plan is the domain type for a developer subscription plan.
type Plan struct {
	ID          string
	DeveloperID string
	Name        string
	Price       int
	Period      string
	Benefits    []string
	ChatLink    string
	Active      bool
	CreatedAt   time.Time
}

// Repo is the persistence port for subscription plans.
type Repo interface {
	Upsert(ctx context.Context, id, devID, name string, price int, period string, benefits []string, chatLink string) (Plan, error)
	GetByDeveloper(ctx context.Context, devID string) (Plan, error)
	GetByID(ctx context.Context, id string) (Plan, error)
	AddGame(ctx context.Context, planID, gameID string) error
	RemoveGame(ctx context.Context, planID, gameID string) error
	ListPlanGameIDs(ctx context.Context, planID string) ([]string, error)
	GetPlanForGameKey(ctx context.Context, gameKey string) (Plan, error)
}

type repo struct{ q *sqlc.Queries }

// NewRepo builds the subscriptions repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapPlan(p sqlc.SubscriptionPlan) Plan {
	return Plan{
		ID:          p.ID,
		DeveloperID: p.DeveloperID,
		Name:        p.Name,
		Price:       int(p.Price),
		Period:      p.Period,
		Benefits:    p.Benefits,
		ChatLink:    p.ChatLink,
		Active:      p.Active,
		CreatedAt:   p.CreatedAt.Time,
	}
}

func (r *repo) Upsert(ctx context.Context, id, devID, name string, price int, period string, benefits []string, chatLink string) (Plan, error) {
	p, err := r.q.UpsertSubscriptionPlan(ctx, sqlc.UpsertSubscriptionPlanParams{
		ID:          id,
		DeveloperID: devID,
		Name:        name,
		Price:       intconv.ToInt32(price),
		Period:      period,
		Benefits:    benefits,
		ChatLink:    chatLink,
	})
	if err != nil {
		return Plan{}, err
	}
	return mapPlan(p), nil
}

func (r *repo) GetByDeveloper(ctx context.Context, devID string) (Plan, error) {
	p, err := r.q.GetPlanByDeveloper(ctx, devID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	return mapPlan(p), nil
}

func (r *repo) GetByID(ctx context.Context, id string) (Plan, error) {
	p, err := r.q.GetPlanByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	return mapPlan(p), nil
}

func (r *repo) AddGame(ctx context.Context, planID, gameID string) error {
	return r.q.AddGameToPlan(ctx, sqlc.AddGameToPlanParams{PlanID: planID, GameID: gameID})
}

func (r *repo) RemoveGame(ctx context.Context, planID, gameID string) error {
	return r.q.RemoveGameFromPlan(ctx, sqlc.RemoveGameFromPlanParams{PlanID: planID, GameID: gameID})
}

func (r *repo) ListPlanGameIDs(ctx context.Context, planID string) ([]string, error) {
	rows, err := r.q.ListPlanGameIDs(ctx, planID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	copy(out, rows)
	return out, nil
}

func (r *repo) GetPlanForGameKey(ctx context.Context, gameKey string) (Plan, error) {
	p, err := r.q.GetPlanForGameKey(ctx, gameKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	return mapPlan(p), nil
}
