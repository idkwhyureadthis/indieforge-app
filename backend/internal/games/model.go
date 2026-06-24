package games

import (
	"strings"
	"time"

	"indieforge/internal/dto"
)

// Subscription is a game's author-defined backer plan.
type Subscription struct {
	Enabled  bool
	Price    int
	Period   string
	Benefits []string
	ChatLink string // perk — never exposed in the public game DTO
}

// DemoDay is a game's free-to-play window, set by its author.
type DemoDay struct {
	Enabled  bool
	StartsAt *time.Time
	EndsAt   *time.Time
}

// Active reports whether the demo window is open right now.
func (d DemoDay) Active() bool {
	if !d.Enabled {
		return false
	}
	now := time.Now()
	if d.StartsAt != nil && now.Before(*d.StartsAt) {
		return false
	}
	if d.EndsAt != nil && now.After(*d.EndsAt) {
		return false
	}
	return true
}

// Game is the rich domain representation used across the games & commerce modules.
type Game struct {
	ID                  string
	Slug                string
	Title               string
	Tagline             string
	Description         string
	Genre               string
	Tags                []string
	DeveloperID         string
	DeveloperName       string
	CoverImage          string
	Screenshots         []string
	HasBrowserBuild     bool
	BrowserBuildURL     string
	HasDownloadBuild    bool
	DownloadObjectKey   string
	DownloadFileName    string
	DownloadSizeMB      int
	DownloadPlatforms   []string
	SupportsMultiplayer bool
	PricingModel        string
	Price               int
	FriendPackDiscount  int
	Subscription        Subscription
	DemoDay             DemoDay
	Theme               dto.Theme
	Status              string
	Plays               int
	CreatedAt           time.Time
}

// FriendPackPrice is the discounted price when gifting via the Friend Pack.
func (g Game) FriendPackPrice() int {
	return int(float64(g.Price) * (1 - float64(g.FriendPackDiscount)/100))
}

// Counts holds the aggregate owner/subscriber numbers for a game.
type Counts struct {
	Owners      int
	Subscribers int
}

// ViewerContext is the per-request acquisition state for the requesting user.
type ViewerContext struct {
	Owned      bool
	Subscribed bool
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 60 {
		out = out[:60]
	}
	if out == "" {
		return "game"
	}
	return out
}
