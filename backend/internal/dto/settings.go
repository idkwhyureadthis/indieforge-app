package dto

// Settings is the runtime service configuration — both the persisted shape
// and the wire shape are identical, so the settings module uses this type
// directly throughout (repo, service, handler).
type Settings struct {
	CommissionPercent int  `json:"commissionPercent"`
	TrendingEnabled   bool `json:"trendingEnabled"`
	PopularEnabled    bool `json:"popularEnabled"`
}
