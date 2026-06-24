package moderation

import (
	"context"
	"errors"

	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
)

// Repo is the persistence port for reports.
type Repo interface {
	Create(ctx context.Context, id, reporterID, targetType, targetID, reason, details string) (Report, error)
	Get(ctx context.Context, id string) (Report, error)
	List(ctx context.Context, status string) ([]Report, error)
	Resolve(ctx context.Context, id, status, resolution, handledBy string) error
}

// Games is the port used to hide/remove reported games.
type Games interface {
	SetStatus(ctx context.Context, id, status string) error
}

// UseCase implements the moderation business rules: filing reports and
// resolving them (dismiss, or hide/remove the reported game).
type UseCase struct {
	repo  Repo
	games Games
}

// NewUseCase wires the moderation usecase to its ports.
func NewUseCase(repo Repo, games Games) *UseCase {
	return &UseCase{repo: repo, games: games}
}

var validReasons = map[string]bool{
	"inappropriate": true, "copyright": true, "broken": true, "scam": true, "other": true,
}

// CreateReport files a new report against a game.
func (uc *UseCase) CreateReport(ctx context.Context, reporter middleware.User, targetType, targetID, reason, details string) (Report, error) {
	if targetType == "" {
		targetType = "game"
	}
	if targetType != "game" {
		return Report{}, apperr.BadRequest("Unsupported report target")
	}
	if targetID == "" {
		return Report{}, apperr.BadRequest("Nothing to report")
	}
	if !validReasons[reason] {
		return Report{}, apperr.BadRequest("Invalid report reason")
	}
	return uc.repo.Create(ctx, idgen.New("rep"), reporter.ID, targetType, targetID, reason, details)
}

// ListReports returns reports filtered by status ("" means all).
func (uc *UseCase) ListReports(ctx context.Context, status string) ([]Report, error) {
	return uc.repo.List(ctx, status)
}

// GetReport fetches a single report by ID.
func (uc *UseCase) GetReport(ctx context.Context, id string) (Report, error) {
	r, err := uc.repo.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Report{}, apperr.NotFound("Report not found")
	}
	return r, err
}

// Resolve applies a moderator decision and, for game actions, updates game status.
func (uc *UseCase) Resolve(ctx context.Context, moderator middleware.User, id, action, note string) (Report, error) {
	report, err := uc.repo.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Report{}, apperr.NotFound("Report not found")
	}
	if err != nil {
		return Report{}, err
	}

	status := "resolved"
	switch action {
	case "dismiss":
		status = "dismissed"
	case "hide-game":
		if err := uc.games.SetStatus(ctx, report.TargetID, "hidden"); err != nil {
			return Report{}, err
		}
	case "remove-game":
		if err := uc.games.SetStatus(ctx, report.TargetID, "removed"); err != nil {
			return Report{}, err
		}
	default:
		return Report{}, apperr.BadRequest("Unknown moderation action")
	}

	if err := uc.repo.Resolve(ctx, id, status, note, moderator.ID); err != nil {
		return Report{}, err
	}
	return uc.repo.Get(ctx, id)
}
