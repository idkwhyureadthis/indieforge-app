package settings

import (
	"context"

	"indieforge/internal/dto"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/intconv"
)

// Repo is the persistence port for settings.
type Repo interface {
	Get(ctx context.Context) (dto.Settings, error)
	Update(ctx context.Context, s dto.Settings) (dto.Settings, error)
}

type repo struct{ q *sqlc.Queries }

// NewRepo builds the settings repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func (r *repo) Get(ctx context.Context) (dto.Settings, error) {
	s, err := r.q.GetSettings(ctx)
	if err != nil {
		return dto.Settings{}, err
	}
	return dto.Settings{
		CommissionPercent: int(s.CommissionPercent),
		TrendingEnabled:   s.TrendingEnabled,
		PopularEnabled:    s.PopularEnabled,
	}, nil
}

func (r *repo) Update(ctx context.Context, s dto.Settings) (dto.Settings, error) {
	out, err := r.q.UpdateSettings(ctx, sqlc.UpdateSettingsParams{
		CommissionPercent: intconv.ToInt32(s.CommissionPercent),
		TrendingEnabled:   s.TrendingEnabled,
		PopularEnabled:    s.PopularEnabled,
	})
	if err != nil {
		return dto.Settings{}, err
	}
	return dto.Settings{
		CommissionPercent: int(out.CommissionPercent),
		TrendingEnabled:   out.TrendingEnabled,
		PopularEnabled:    out.PopularEnabled,
	}, nil
}

// Service holds settings business logic.
type Service struct{ repo Repo }

// NewService wires the settings service to its repo.
func NewService(repo Repo) *Service { return &Service{repo: repo} }

// Get returns the current service settings.
func (s *Service) Get(ctx context.Context) (dto.Settings, error) { return s.repo.Get(ctx) }

// Update validates and persists new service settings.
func (s *Service) Update(ctx context.Context, in dto.Settings) (dto.Settings, error) {
	if in.CommissionPercent < 0 {
		in.CommissionPercent = 0
	}
	if in.CommissionPercent > 100 {
		in.CommissionPercent = 100
	}
	return s.repo.Update(ctx, in)
}

// HomeFlags satisfies the games.Settings port.
func (s *Service) HomeFlags(ctx context.Context) (trending, popular bool, err error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return false, false, err
	}
	return st.TrendingEnabled, st.PopularEnabled, nil
}

// Commission returns the current commission percent.
func (s *Service) Commission(ctx context.Context) (int, error) {
	st, err := s.repo.Get(ctx)
	return st.CommissionPercent, err
}
