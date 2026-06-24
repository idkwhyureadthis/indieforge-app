package moderation

import "indieforge/internal/dto"

func toDTO(r Report) dto.ReportDTO {
	return dto.ReportDTO{
		ID:         r.ID,
		ReporterID: r.ReporterID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Reason:     r.Reason,
		Details:    r.Details,
		Status:     r.Status,
		Resolution: r.Resolution,
		HandledBy:  r.HandledBy,
		CreatedAt:  dto.FormatTime(r.CreatedAt),
		ResolvedAt: dto.FormatTimePtr(r.ResolvedAt),
	}
}
