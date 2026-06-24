package games

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
	"indieforge/pkg/intconv"
)

// Repo is the persistence port for the games usecase.
type Repo interface {
	Create(ctx context.Context, p sqlc.CreateGameParams) (Game, error)
	GetByID(ctx context.Context, id string) (Game, error)
	GetBySlug(ctx context.Context, slug string) (Game, error)
	ListPublished(ctx context.Context) ([]Game, error)
	ListByDeveloper(ctx context.Context, developerID string) ([]Game, error)
	ListNewest(ctx context.Context, limit int) ([]Game, error)
	ListTrending(ctx context.Context, limit int) ([]Game, error)
	ListPopular(ctx context.Context, limit int) ([]Game, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	SetStatus(ctx context.Context, id, status string) error
	InsertEvent(ctx context.Context, id, gameID, eventType string) error
	RecomputeTrending(ctx context.Context) error
	OwnerCounts(ctx context.Context) (map[string]int, error)
	SubscriberCounts(ctx context.Context) (map[string]int, error)
	CountOwners(ctx context.Context, gameID string) (int, error)
	CountSubscribers(ctx context.Context, gameID string) (int, error)
	HasOwnership(ctx context.Context, userID, gameID string) (bool, error)
	HasSubscription(ctx context.Context, userID, gameID string) (bool, error)
	OwnedGameIDs(ctx context.Context, userID string) (map[string]bool, error)
	SubscribedGameIDs(ctx context.Context, userID string) (map[string]bool, error)
}

// Storage is the object-storage port (S3/MinIO).
type Storage interface {
	PutPublic(ctx context.Context, key, contentType string, data []byte) (string, error)
	PutPrivate(ctx context.Context, key, contentType string, data []byte) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	ExtractZipToPrefix(ctx context.Context, prefix string, zipData []byte) (string, error)
}

// Scanner is the antivirus port.
type Scanner interface {
	Scan(ctx context.Context, r io.Reader) (bool, string, error)
}

// UseCase implements the games business rules: catalog browsing, upload and
// publishing, the curated home sections, and presigned downloads.
type UseCase struct {
	repo    Repo
	store   Storage
	scanner Scanner
}

// NewUseCase wires the games usecase to its ports.
func NewUseCase(repo Repo, store Storage, scanner Scanner) *UseCase {
	return &UseCase{repo: repo, store: store, scanner: scanner}
}

// Upload is one in-memory uploaded file.
type Upload struct {
	Filename    string
	ContentType string
	Data        []byte
}

// NewGame is the validated input for creating a game.
type NewGame struct {
	Title               string
	Tagline             string
	Description         string
	Genre               string
	Tags                []string
	HasBrowserBuild     bool
	BrowserBuildURL     string
	HasDownloadBuild    bool
	DownloadPlatforms   []string
	SupportsMultiplayer bool
	PricingModel        string
	Price               int
	FriendPackDiscount  int
	Subscription        Subscription
	DemoDay             DemoDay
	Theme               dto.Theme
	Cover               *Upload
	Screenshots         []*Upload
	BrowserBuildZip     *Upload
	DownloadFile        *Upload
}

// Filters narrows GET /games to a search, genre, tag, pricing model, and sort order.
type Filters struct {
	Search  string
	Genre   string
	Tag     string
	Pricing string // free | paid | subscription | demo
	Sort    string // new | popular | price-asc | price-desc
}

// ---- reads --------------------------------------------------------------

// GameByKey resolves a game by slug, falling back to ID.
func (uc *UseCase) GameByKey(ctx context.Context, key string) (Game, error) {
	g, err := uc.repo.GetBySlug(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return uc.repo.GetByID(ctx, key)
	}
	return g, err
}

// RecordEvent appends a game activity event (view | play | acquire).
func (uc *UseCase) RecordEvent(ctx context.Context, gameID, eventType string) error {
	return uc.repo.InsertEvent(ctx, idgen.New("evt"), gameID, eventType)
}

// Get returns a serialized game (by id or slug) and records a view.
func (uc *UseCase) Get(ctx context.Context, key, viewerID string) (dto.GameDTO, error) {
	g, err := uc.GameByKey(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return dto.GameDTO{}, apperr.NotFound("Game not found")
	}
	if err != nil {
		return dto.GameDTO{}, err
	}
	_ = uc.RecordEvent(ctx, g.ID, "view")
	return uc.Serialize(ctx, g, viewerID)
}

// Serialize converts a domain game into the public DTO for a given viewer.
func (uc *UseCase) Serialize(ctx context.Context, g Game, viewerID string) (dto.GameDTO, error) {
	owners, err := uc.repo.CountOwners(ctx, g.ID)
	if err != nil {
		return dto.GameDTO{}, err
	}
	subs, err := uc.repo.CountSubscribers(ctx, g.ID)
	if err != nil {
		return dto.GameDTO{}, err
	}
	v := ViewerContext{}
	if viewerID != "" {
		if v.Owned, err = uc.repo.HasOwnership(ctx, viewerID, g.ID); err != nil {
			return dto.GameDTO{}, err
		}
		if v.Subscribed, err = uc.repo.HasSubscription(ctx, viewerID, g.ID); err != nil {
			return dto.GameDTO{}, err
		}
	}
	return toDTO(g, Counts{Owners: owners, Subscribers: subs}, v), nil
}

// List returns published games matching f, serialized for viewerID.
func (uc *UseCase) List(ctx context.Context, f Filters, viewerID string) ([]dto.GameDTO, error) {
	all, err := uc.repo.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	all = applyFilters(all, f)
	return uc.serializeMany(ctx, all, viewerID, f.Sort)
}

// MyGames returns every game (any status) created by developerID.
func (uc *UseCase) MyGames(ctx context.Context, developerID string) ([]dto.GameDTO, error) {
	list, err := uc.repo.ListByDeveloper(ctx, developerID)
	if err != nil {
		return nil, err
	}
	return uc.serializeMany(ctx, list, developerID, "new")
}

// Home builds the curated home-page sections. Trending/Popular are computed
// only when their flag is enabled; Newest and DemoDay are always populated.
func (uc *UseCase) Home(ctx context.Context, viewerID string, trendingEnabled, popularEnabled bool, limit int) (dto.HomeSections, error) {
	// Default disabled/empty sections to [] rather than the zero-value nil
	// slice, since nil marshals to JSON null and the frontend calls array
	// methods (e.g. .length) on these unconditionally.
	out := dto.HomeSections{Trending: []dto.GameDTO{}, Popular: []dto.GameDTO{}}
	newest, err := uc.repo.ListNewest(ctx, limit)
	if err != nil {
		return out, err
	}
	if out.Newest, err = uc.serializeMany(ctx, newest, viewerID, "new"); err != nil {
		return out, err
	}
	// Demo Day = currently active demos among the newest set.
	var demos []Game
	for _, g := range newest {
		if g.DemoDay.Active() {
			demos = append(demos, g)
		}
	}
	if out.DemoDay, err = uc.serializeMany(ctx, demos, viewerID, "new"); err != nil {
		return out, err
	}
	if trendingEnabled {
		t, err := uc.repo.ListTrending(ctx, limit)
		if err != nil {
			return out, err
		}
		if out.Trending, err = uc.serializeMany(ctx, t, viewerID, ""); err != nil {
			return out, err
		}
	}
	if popularEnabled {
		p, err := uc.repo.ListPopular(ctx, limit)
		if err != nil {
			return out, err
		}
		if out.Popular, err = uc.serializeMany(ctx, p, viewerID, ""); err != nil {
			return out, err
		}
	}
	return out, nil
}

// DownloadURL issues a presigned URL for an owner of the game.
func (uc *UseCase) DownloadURL(ctx context.Context, key, viewerID string) (string, error) {
	g, err := uc.GameByKey(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return "", apperr.NotFound("Game not found")
	}
	if err != nil {
		return "", err
	}
	if !g.HasDownloadBuild || g.DownloadObjectKey == "" {
		return "", apperr.BadRequest("This game has no downloadable build")
	}
	owned, err := uc.repo.HasOwnership(ctx, viewerID, g.ID)
	if err != nil {
		return "", err
	}
	isAuthor := viewerID != "" && viewerID == g.DeveloperID
	if !owned && !isAuthor && g.PricingModel != "free" && !g.DemoDay.Active() {
		return "", apperr.Forbidden("You need to own this game to download it")
	}
	return uc.store.PresignGet(ctx, g.DownloadObjectKey, 5*time.Minute)
}

// RecomputeTrending refreshes every game's trending score from recent
// activity. Called periodically by a background ticker in main.go.
func (uc *UseCase) RecomputeTrending(ctx context.Context) error {
	return uc.repo.RecomputeTrending(ctx)
}

// SetStatus changes a game's moderation status (used by the moderation module).
func (uc *UseCase) SetStatus(ctx context.Context, id, status string) error {
	return uc.repo.SetStatus(ctx, id, status)
}

// Create validates and publishes a new game: scans every uploaded file,
// stores cover/screenshots/browser-build/download in object storage, and
// persists the listing.
func (uc *UseCase) Create(ctx context.Context, dev middleware.User, in NewGame) (dto.GameDTO, error) {
	if strings.TrimSpace(in.Title) == "" {
		return dto.GameDTO{}, apperr.BadRequest("Give your game a title")
	}
	if !in.HasBrowserBuild && !in.HasDownloadBuild {
		return dto.GameDTO{}, apperr.BadRequest("Add at least one build: browser or downloadable")
	}

	// Antivirus: scan every uploaded file before anything is stored.
	uploads := []*Upload{in.Cover, in.BrowserBuildZip, in.DownloadFile}
	uploads = append(uploads, in.Screenshots...)
	for _, u := range uploads {
		if u == nil {
			continue
		}
		clean, sig, err := uc.scanner.Scan(ctx, bytes.NewReader(u.Data))
		if err != nil {
			return dto.GameDTO{}, apperr.Internal("Virus scan failed")
		}
		if !clean {
			return dto.GameDTO{}, apperr.Unprocessable(fmt.Sprintf("File %q rejected by antivirus (%s)", u.Filename, sig))
		}
	}

	id := idgen.New("game")
	slug, err := uc.uniqueSlug(ctx, in.Title)
	if err != nil {
		return dto.GameDTO{}, err
	}

	coverURL := ""
	if in.Cover != nil {
		coverURL, err = uc.store.PutPublic(ctx, "media/"+id+"/cover"+ext(in.Cover.Filename), in.Cover.ContentType, in.Cover.Data)
		if err != nil {
			return dto.GameDTO{}, apperr.Internal("Could not store cover image")
		}
	}
	shots := make([]string, 0, len(in.Screenshots))
	for i, s := range in.Screenshots {
		url, err := uc.store.PutPublic(ctx, fmt.Sprintf("media/%s/shot-%d%s", id, i, ext(s.Filename)), s.ContentType, s.Data)
		if err != nil {
			return dto.GameDTO{}, apperr.Internal("Could not store screenshot")
		}
		shots = append(shots, url)
	}

	browserURL := strings.TrimSpace(in.BrowserBuildURL)
	if in.BrowserBuildZip != nil {
		browserURL, err = uc.store.ExtractZipToPrefix(ctx, "web/"+id, in.BrowserBuildZip.Data)
		if err != nil {
			return dto.GameDTO{}, apperr.Unprocessable("Could not unpack browser build: " + err.Error())
		}
	}
	hasBrowser := browserURL != ""

	dlKey, dlName, dlSize := "", "", 0
	if in.DownloadFile != nil {
		dlName = in.DownloadFile.Filename
		dlKey = "downloads/" + id + "/" + dlName
		if err := uc.store.PutPrivate(ctx, dlKey, in.DownloadFile.ContentType, in.DownloadFile.Data); err != nil {
			return dto.GameDTO{}, apperr.Internal("Could not store build file")
		}
		dlSize = len(in.DownloadFile.Data) / (1024 * 1024)
		if dlSize < 1 {
			dlSize = 1
		}
	}
	hasDownload := dlKey != ""

	if !hasBrowser && !hasDownload {
		return dto.GameDTO{}, apperr.BadRequest("Provide a browser build URL/zip or a downloadable file")
	}

	pricing := "free"
	price := 0
	if in.PricingModel == "paid" {
		pricing = "paid"
		if in.Price > 0 {
			price = in.Price
		}
	}

	themeJSON, _ := json.Marshal(in.Theme)
	subBenefits := in.Subscription.Benefits
	if !in.Subscription.Enabled {
		subBenefits = []string{}
	}

	g, err := uc.repo.Create(ctx, sqlc.CreateGameParams{
		ID:                  id,
		Slug:                slug,
		Title:               strings.TrimSpace(in.Title),
		Tagline:             strings.TrimSpace(in.Tagline),
		Description:         strings.TrimSpace(in.Description),
		Genre:               orDefault(in.Genre, "Other"),
		Tags:                in.Tags,
		DeveloperID:         dev.ID,
		DeveloperName:       dev.Username,
		CoverImage:          coverURL,
		Screenshots:         shots,
		HasBrowserBuild:     hasBrowser,
		BrowserBuildUrl:     browserURL,
		HasDownloadBuild:    hasDownload,
		DownloadObjectKey:   dlKey,
		DownloadFileName:    dlName,
		DownloadSizeMb:      intconv.ToInt32(dlSize),
		DownloadPlatforms:   in.DownloadPlatforms,
		SupportsMultiplayer: in.SupportsMultiplayer,
		PricingModel:        pricing,
		Price:               intconv.ToInt32(price),
		FriendPackDiscount:  intconv.ToInt32(clamp(in.FriendPackDiscount, 0, 90)),
		SubEnabled:          in.Subscription.Enabled,
		SubPrice:            intconv.ToInt32(in.Subscription.Price),
		SubBenefits:         subBenefits,
		SubChatLink:         in.Subscription.ChatLink,
		DemoEnabled:         in.DemoDay.Enabled,
		DemoStartsAt:        in.DemoDay.StartsAt,
		DemoEndsAt:          in.DemoDay.EndsAt,
		Theme:               themeJSON,
		Status:              "published",
	})
	if err != nil {
		return dto.GameDTO{}, err
	}
	return uc.Serialize(ctx, g, dev.ID)
}

// ---- helpers ------------------------------------------------------------

func (uc *UseCase) uniqueSlug(ctx context.Context, title string) (string, error) {
	base := slugify(title)
	slug := base
	for n := 2; ; n++ {
		exists, err := uc.repo.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
}

func (uc *UseCase) serializeMany(ctx context.Context, list []Game, viewerID, sortKey string) ([]dto.GameDTO, error) {
	if len(list) == 0 {
		return []dto.GameDTO{}, nil
	}
	ownerCounts, err := uc.repo.OwnerCounts(ctx)
	if err != nil {
		return nil, err
	}
	subCounts, err := uc.repo.SubscriberCounts(ctx)
	if err != nil {
		return nil, err
	}
	var ownedSet, subSet map[string]bool
	if viewerID != "" {
		if ownedSet, err = uc.repo.OwnedGameIDs(ctx, viewerID); err != nil {
			return nil, err
		}
		if subSet, err = uc.repo.SubscribedGameIDs(ctx, viewerID); err != nil {
			return nil, err
		}
	}
	out := make([]dto.GameDTO, 0, len(list))
	for _, g := range list {
		c := Counts{Owners: ownerCounts[g.ID], Subscribers: subCounts[g.ID]}
		v := ViewerContext{Owned: ownedSet[g.ID], Subscribed: subSet[g.ID]}
		out = append(out, toDTO(g, c, v))
	}
	sortDTOs(out, sortKey)
	return out, nil
}

func applyFilters(list []Game, f Filters) []Game {
	q := strings.ToLower(strings.TrimSpace(f.Search))
	out := list[:0]
	for _, g := range list {
		if q != "" &&
			!strings.Contains(strings.ToLower(g.Title), q) &&
			!strings.Contains(strings.ToLower(g.Tagline), q) &&
			!strings.Contains(strings.ToLower(g.DeveloperName), q) &&
			!containsTag(g.Tags, q) {
			continue
		}
		if f.Genre != "" && g.Genre != f.Genre {
			continue
		}
		if f.Tag != "" && !containsTag(g.Tags, f.Tag) {
			continue
		}
		switch f.Pricing {
		case "free":
			if g.PricingModel != "free" {
				continue
			}
		case "paid":
			if g.PricingModel != "paid" {
				continue
			}
		case "subscription":
			if !g.Subscription.Enabled {
				continue
			}
		case "demo":
			if !g.DemoDay.Active() {
				continue
			}
		}
		out = append(out, g)
	}
	return out
}

func sortDTOs(list []dto.GameDTO, sortKey string) {
	switch sortKey {
	case "popular":
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Stats.Owners+list[i].Stats.Subscribers > list[j].Stats.Owners+list[j].Stats.Subscribers
		})
	case "price-asc":
		sort.SliceStable(list, func(i, j int) bool { return list[i].Price < list[j].Price })
	case "price-desc":
		sort.SliceStable(list, func(i, j int) bool { return list[i].Price > list[j].Price })
	}
}

func containsTag(tags []string, t string) bool {
	for _, x := range tags {
		if strings.Contains(strings.ToLower(x), t) {
			return true
		}
	}
	return false
}

func ext(name string) string { return filepath.Ext(name) }

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
