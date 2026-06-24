package dto

// ReportDTO is the public view of a moderation report.
type ReportDTO struct {
	ID         string  `json:"id"`
	ReporterID string  `json:"reporterId"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Reason     string  `json:"reason"`
	Details    string  `json:"details"`
	Status     string  `json:"status"`
	Resolution string  `json:"resolution"`
	HandledBy  *string `json:"handledBy"`
	CreatedAt  string  `json:"createdAt"`
	ResolvedAt *string `json:"resolvedAt"`
}

// CreateReportRequest is the POST /reports request body.
type CreateReportRequest struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

// ResolveRequest is the POST /moderation/reports/{id}/resolve request body.
type ResolveRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}
