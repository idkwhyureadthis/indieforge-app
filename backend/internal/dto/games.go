package dto

// Theme is the itch.io-style per-game page customisation. Shared as-is
// between the create-game input and the public game DTO (pure data, no
// internal-only fields), so games keeps a single definition here.
type Theme struct {
	Accent     string `json:"accent"`
	Accent2    string `json:"accent2"`
	Background string `json:"background"`
	Layout     string `json:"layout"`
	CardShape  string `json:"cardShape"`
}

// SubscriptionDTO is the public view of a game's author-defined subscription
// plan. The ChatLink perk is deliberately absent — it's only ever returned by
// the perks endpoint to an active subscriber or the author.
type SubscriptionDTO struct {
	Enabled  bool     `json:"enabled"`
	Price    int      `json:"price"`
	Period   string   `json:"period"`
	Benefits []string `json:"benefits"`
}

// DemoDayDTO is the public view of a game's free-to-play window.
type DemoDayDTO struct {
	Enabled  bool    `json:"enabled"`
	StartsAt *string `json:"startsAt"`
	EndsAt   *string `json:"endsAt"`
	Active   bool    `json:"active"`
}

// StatsDTO holds a game's public aggregate counters.
type StatsDTO struct {
	Owners      int `json:"owners"`
	Subscribers int `json:"subscribers"`
	Plays       int `json:"plays"`
}

// GameDTO mirrors the frontend Game type (camelCase, nullable fields as null).
type GameDTO struct {
	ID                  string          `json:"id"`
	Slug                string          `json:"slug"`
	Title               string          `json:"title"`
	Tagline             string          `json:"tagline"`
	Description         string          `json:"description"`
	Genre               string          `json:"genre"`
	Tags                []string        `json:"tags"`
	DeveloperID         string          `json:"developerId"`
	DeveloperName       string          `json:"developerName"`
	CoverImage          *string         `json:"coverImage"`
	Screenshots         []string        `json:"screenshots"`
	HasBrowserBuild     bool            `json:"hasBrowserBuild"`
	BrowserBuildURL     *string         `json:"browserBuildUrl"`
	HasDownloadBuild    bool            `json:"hasDownloadBuild"`
	DownloadFileName    *string         `json:"downloadFileName"`
	DownloadSizeMB      *int            `json:"downloadSizeMB"`
	DownloadPlatforms   []string        `json:"downloadPlatforms"`
	SupportsMultiplayer bool            `json:"supportsMultiplayer"`
	PricingModel        string          `json:"pricingModel"`
	Price               int             `json:"price"`
	FriendPackDiscount  int             `json:"friendPackDiscount"`
	Subscription        SubscriptionDTO `json:"subscription"`
	DemoDay             DemoDayDTO      `json:"demoDay"`
	Theme               Theme           `json:"theme"`
	Status              string          `json:"status"`
	CreatedAt           string          `json:"createdAt"`
	Stats               StatsDTO        `json:"stats"`
	Owned               bool            `json:"owned"`
	Subscribed          bool            `json:"subscribed"`
	CanPlayFree         bool            `json:"canPlayFree"`
}

// HomeSections holds the curated home-page lists.
type HomeSections struct {
	Trending []GameDTO `json:"trending"`
	Popular  []GameDTO `json:"popular"`
	Newest   []GameDTO `json:"newest"`
	DemoDay  []GameDTO `json:"demoDay"`
}
