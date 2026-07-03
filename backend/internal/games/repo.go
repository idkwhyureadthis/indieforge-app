package games

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/dto"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/intconv"
)

// ErrNotFound is returned when a game row is absent.
var ErrNotFound = errors.New("game not found")

type repo struct{ q *sqlc.Queries }

// NewRepo builds the games repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapGame(g sqlc.Game) Game {
	var theme dto.Theme
	if len(g.Theme) > 0 {
		_ = json.Unmarshal(g.Theme, &theme)
	}
	return Game{
		ID:                  g.ID,
		Slug:                g.Slug,
		Title:               g.Title,
		Tagline:             g.Tagline,
		Description:         g.Description,
		Genre:               g.Genre,
		Tags:                g.Tags,
		DeveloperID:         g.DeveloperID,
		DeveloperName:       g.DeveloperName,
		CoverImage:          g.CoverImage,
		Screenshots:         g.Screenshots,
		HasBrowserBuild:     g.HasBrowserBuild,
		BrowserBuildURL:     g.BrowserBuildUrl,
		HasDownloadBuild:    g.HasDownloadBuild,
		DownloadObjectKey:   g.DownloadObjectKey,
		DownloadFileName:    g.DownloadFileName,
		DownloadSizeMB:      int(g.DownloadSizeMb),
		DownloadPlatforms:   g.DownloadPlatforms,
		SupportsMultiplayer: g.SupportsMultiplayer,
		PricingModel:        g.PricingModel,
		Price:               int(g.Price),
		FriendPackDiscount:  int(g.FriendPackDiscount),
		Subscription: Subscription{
			Enabled:  g.SubEnabled,
			Price:    int(g.SubPrice),
			Period:   "month",
			Benefits: g.SubBenefits,
			ChatLink: g.SubChatLink,
		},
		DemoDay:   DemoDay{Enabled: g.DemoEnabled, StartsAt: g.DemoStartsAt, EndsAt: g.DemoEndsAt},
		Theme:     theme,
		Status:    g.Status,
		Plays:     int(g.Plays),
		CreatedAt: g.CreatedAt.Time,
	}
}

func mapGames(rows []sqlc.Game) []Game {
	out := make([]Game, len(rows))
	for i, r := range rows {
		out[i] = mapGame(r)
	}
	return out
}

func (r *repo) Create(ctx context.Context, p sqlc.CreateGameParams) (Game, error) {
	g, err := r.q.CreateGame(ctx, p)
	if err != nil {
		return Game{}, err
	}
	return mapGame(g), nil
}

func (r *repo) GetByID(ctx context.Context, id string) (Game, error) {
	g, err := r.q.GetGameByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, err
	}
	return mapGame(g), nil
}

func (r *repo) GetBySlug(ctx context.Context, slug string) (Game, error) {
	g, err := r.q.GetGameBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, err
	}
	return mapGame(g), nil
}

func (r *repo) ListPublished(ctx context.Context) ([]Game, error) {
	rows, err := r.q.ListPublishedGames(ctx)
	return mapGames(rows), err
}

func (r *repo) ListByDeveloper(ctx context.Context, developerID string) ([]Game, error) {
	rows, err := r.q.ListGamesByDeveloper(ctx, developerID)
	return mapGames(rows), err
}

func (r *repo) ListNewest(ctx context.Context, limit int) ([]Game, error) {
	rows, err := r.q.ListNewest(ctx, intconv.ToInt32(limit))
	return mapGames(rows), err
}

func (r *repo) ListTrending(ctx context.Context, limit int) ([]Game, error) {
	rows, err := r.q.ListTrending(ctx, intconv.ToInt32(limit))
	return mapGames(rows), err
}

func (r *repo) ListPopular(ctx context.Context, limit int) ([]Game, error) {
	rows, err := r.q.ListPopular(ctx, intconv.ToInt32(limit))
	return mapGames(rows), err
}

func (r *repo) SlugExists(ctx context.Context, slug string) (bool, error) {
	return r.q.SlugExists(ctx, slug)
}

func (r *repo) SetStatus(ctx context.Context, id, status string) error {
	return r.q.SetGameStatus(ctx, sqlc.SetGameStatusParams{ID: id, Status: status})
}

func (r *repo) InsertEvent(ctx context.Context, id, gameID, eventType string) error {
	return r.q.InsertGameEvent(ctx, sqlc.InsertGameEventParams{ID: id, GameID: gameID, Type: eventType})
}

func (r *repo) IncrementPlays(ctx context.Context, id string) error {
	return r.q.IncrementPlays(ctx, id)
}

func (r *repo) RecomputeTrending(ctx context.Context) error {
	return r.q.RecomputeTrendingScores(ctx)
}

// OwnerCounts returns a map of gameID -> owner count.
func (r *repo) OwnerCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.q.OwnerCounts(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(rows))
	for _, row := range rows {
		m[row.GameID] = int(row.N)
	}
	return m, nil
}

// SubscriberCounts returns a map of gameID -> active-subscriber count.
func (r *repo) SubscriberCounts(ctx context.Context) (map[string]int, error) {
	rows, err := r.q.SubscriberCounts(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(rows))
	for _, row := range rows {
		m[row.GameID] = int(row.N)
	}
	return m, nil
}

func (r *repo) CountOwners(ctx context.Context, gameID string) (int, error) {
	n, err := r.q.CountOwnersByGame(ctx, gameID)
	return int(n), err
}

func (r *repo) CountSubscribers(ctx context.Context, gameID string) (int, error) {
	n, err := r.q.CountSubscribersByGame(ctx, gameID)
	return int(n), err
}

func (r *repo) HasOwnership(ctx context.Context, userID, gameID string) (bool, error) {
	_, err := r.q.GetOwnership(ctx, sqlc.GetOwnershipParams{UserID: userID, GameID: gameID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *repo) HasSubscription(ctx context.Context, userID, gameID string) (bool, error) {
	_, err := r.q.GetActiveSubscription(ctx, sqlc.GetActiveSubscriptionParams{UserID: userID, GameID: gameID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// OwnedGameIDs returns the set of game IDs the user owns.
func (r *repo) OwnedGameIDs(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.q.ListOwnershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rows))
	for _, o := range rows {
		set[o.GameID] = true
	}
	return set, nil
}

// SubscribedGameIDs returns the set of game IDs the user is actively subscribed to.
func (r *repo) SubscribedGameIDs(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.q.ListSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rows))
	for _, s := range rows {
		set[s.GameID] = true
	}
	return set, nil
}

func (r *repo) MarkDeveloper(ctx context.Context, userID string) error {
	return r.q.MarkDeveloper(ctx, userID)
}
