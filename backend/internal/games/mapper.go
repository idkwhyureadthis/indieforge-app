package games

import "indieforge/internal/dto"

// toDTO maps the rich domain Game (plus per-request aggregates) to its wire
// representation. The DTO shape itself lives in internal/dto since it's pure
// data; the mapping stays here because it needs games-internal types
// (Game, Counts, ViewerContext).
func toDTO(g Game, c Counts, v ViewerContext) dto.GameDTO {
	demoActive := g.DemoDay.Active()
	canPlayFree := g.PricingModel == "free" || v.Owned || v.Subscribed || demoActive

	var sizePtr *int
	if g.HasDownloadBuild && g.DownloadSizeMB > 0 {
		sz := g.DownloadSizeMB
		sizePtr = &sz
	}

	return dto.GameDTO{
		ID:                  g.ID,
		Slug:                g.Slug,
		Title:               g.Title,
		Tagline:             g.Tagline,
		Description:         g.Description,
		Genre:               g.Genre,
		Tags:                dto.NonNilStrings(g.Tags),
		DeveloperID:         g.DeveloperID,
		DeveloperName:       g.DeveloperName,
		CoverImage:          dto.StrPtr(g.CoverImage),
		Screenshots:         dto.NonNilStrings(g.Screenshots),
		HasBrowserBuild:     g.HasBrowserBuild,
		BrowserBuildURL:     dto.StrPtr(g.BrowserBuildURL),
		HasDownloadBuild:    g.HasDownloadBuild,
		DownloadFileName:    dto.StrPtr(g.DownloadFileName),
		DownloadSizeMB:      sizePtr,
		DownloadPlatforms:   dto.NonNilStrings(g.DownloadPlatforms),
		SupportsMultiplayer: g.SupportsMultiplayer,
		PricingModel:        g.PricingModel,
		Price:               g.Price,
		FriendPackDiscount:  g.FriendPackDiscount,
		Subscription: dto.SubscriptionDTO{
			Enabled:  g.Subscription.Enabled,
			Price:    g.Subscription.Price,
			Period:   "month",
			Benefits: dto.NonNilStrings(g.Subscription.Benefits),
		},
		DemoDay: dto.DemoDayDTO{
			Enabled:  g.DemoDay.Enabled,
			StartsAt: dto.FormatTimePtr(g.DemoDay.StartsAt),
			EndsAt:   dto.FormatTimePtr(g.DemoDay.EndsAt),
			Active:   demoActive,
		},
		Theme:       g.Theme,
		Status:      g.Status,
		CreatedAt:   dto.FormatTime(g.CreatedAt),
		Stats:       dto.StatsDTO{Owners: c.Owners, Subscribers: c.Subscribers, Plays: g.Plays},
		Owned:       v.Owned,
		Subscribed:  v.Subscribed,
		CanPlayFree: canPlayFree,
	}
}
