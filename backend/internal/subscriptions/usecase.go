package subscriptions

import (
	"context"
	"errors"

	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
)

// GamesReader is the slice of the games module this usecase depends on.
type GamesReader interface {
	GameByKey(ctx context.Context, key string) (games.Game, error)
	ListByDeveloper(ctx context.Context, developerID string) ([]games.Game, error)
}

// UseCase implements the subscription plan business rules.
type UseCase struct {
	repo  Repo
	games GamesReader
}

// NewUseCase wires the subscriptions usecase.
func NewUseCase(repo Repo, games GamesReader) *UseCase {
	return &UseCase{repo: repo, games: games}
}

// PlanWithGames carries a plan and its included games.
type PlanWithGames struct {
	Plan
	Games []games.Game
}

func (uc *UseCase) enrichPlan(ctx context.Context, plan Plan) (PlanWithGames, error) {
	ids, err := uc.repo.ListPlanGameIDs(ctx, plan.ID)
	if err != nil {
		return PlanWithGames{}, err
	}
	gs := make([]games.Game, 0, len(ids))
	for _, id := range ids {
		g, err := uc.games.GameByKey(ctx, id)
		if errors.Is(err, games.ErrNotFound) {
			continue
		}
		if err != nil {
			return PlanWithGames{}, err
		}
		gs = append(gs, g)
	}
	return PlanWithGames{Plan: plan, Games: gs}, nil
}

// MyPlan returns the caller's subscription plan (or ErrNotFound if none).
func (uc *UseCase) MyPlan(ctx context.Context, user middleware.User) (PlanWithGames, error) {
	plan, err := uc.repo.GetByDeveloper(ctx, user.ID)
	if errors.Is(err, ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("No subscription plan yet")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// UpsertPlan creates or updates the caller's subscription plan.
func (uc *UseCase) UpsertPlan(ctx context.Context, user middleware.User, name string, price int, period string, benefits []string, chatLink string) (PlanWithGames, error) {
	if !user.IsDeveloper {
		return PlanWithGames{}, apperr.Forbidden("Only developers can manage subscription plans")
	}
	if price <= 0 {
		return PlanWithGames{}, apperr.BadRequest("Price must be greater than zero")
	}
	existing, err := uc.repo.GetByDeveloper(ctx, user.ID)
	var planID string
	switch {
	case errors.Is(err, ErrNotFound):
		planID = idgen.New("plan")

	case err != nil:
		return PlanWithGames{}, err

	default:
		planID = existing.ID
	}
	plan, err := uc.repo.Upsert(ctx, planID, user.ID, name, price, period, benefits, chatLink)
	if err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// AddGame adds one of the developer's games to their subscription plan.
func (uc *UseCase) AddGame(ctx context.Context, user middleware.User, gameKey string) (PlanWithGames, error) {
	plan, err := uc.repo.GetByDeveloper(ctx, user.ID)
	if errors.Is(err, ErrNotFound) {
		return PlanWithGames{}, apperr.BadRequest("Create a subscription plan first")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	g, err := uc.games.GameByKey(ctx, gameKey)
	if errors.Is(err, games.ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("Game not found")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	if g.DeveloperID != user.ID {
		return PlanWithGames{}, apperr.Forbidden("You can only add your own games to your plan")
	}
	if err := uc.repo.AddGame(ctx, plan.ID, g.ID); err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// RemoveGame removes a game from the developer's subscription plan.
func (uc *UseCase) RemoveGame(ctx context.Context, user middleware.User, gameKey string) (PlanWithGames, error) {
	plan, err := uc.repo.GetByDeveloper(ctx, user.ID)
	if errors.Is(err, ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("No subscription plan")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	g, err := uc.games.GameByKey(ctx, gameKey)
	if errors.Is(err, games.ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("Game not found")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	if err := uc.repo.RemoveGame(ctx, plan.ID, g.ID); err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// PlanForGame returns the subscription plan for the developer of the given game (public).
func (uc *UseCase) PlanForGame(ctx context.Context, gameKey string) (PlanWithGames, error) {
	plan, err := uc.repo.GetPlanForGameKey(ctx, gameKey)
	if errors.Is(err, ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("No subscription plan for this game")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// GetPlanByID returns a plan by ID (used during checkout).
func (uc *UseCase) GetPlanByID(ctx context.Context, id string) (PlanWithGames, error) {
	plan, err := uc.repo.GetByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return PlanWithGames{}, apperr.NotFound("Subscription plan not found")
	}
	if err != nil {
		return PlanWithGames{}, err
	}
	return uc.enrichPlan(ctx, plan)
}

// MyGames returns all the caller's games (for the "add game" dropdown).
func (uc *UseCase) MyGames(ctx context.Context, user middleware.User) ([]games.Game, error) {
	return uc.games.ListByDeveloper(ctx, user.ID)
}
