package moderation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/platform/db/sqlc"
)

// ErrNotFound is returned when a report row is absent.
var ErrNotFound = errors.New("report not found")

// Report is a user-filed complaint against a game.
type Report struct {
	ID         string
	ReporterID string
	TargetType string
	TargetID   string
	Reason     string
	Details    string
	Status     string
	Resolution string
	HandledBy  *string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

type repo struct{ q *sqlc.Queries }

// NewRepo builds the moderation repository over the sqlc queries.
func NewRepo(q *sqlc.Queries) Repo { return &repo{q: q} }

func mapReport(r sqlc.Report) Report {
	return Report{
		ID:         r.ID,
		ReporterID: r.ReporterID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Reason:     r.Reason,
		Details:    r.Details,
		Status:     r.Status,
		Resolution: r.Resolution,
		HandledBy:  r.HandledBy,
		CreatedAt:  r.CreatedAt.Time,
		ResolvedAt: r.ResolvedAt,
	}
}

func mapReports(rows []sqlc.Report) []Report {
	out := make([]Report, len(rows))
	for i, r := range rows {
		out[i] = mapReport(r)
	}
	return out
}

func (r *repo) Create(ctx context.Context, id, reporterID, targetType, targetID, reason, details string) (Report, error) {
	rep, err := r.q.CreateReport(ctx, sqlc.CreateReportParams{
		ID: id, ReporterID: reporterID, TargetType: targetType, TargetID: targetID, Reason: reason, Details: details,
	})
	if err != nil {
		return Report{}, err
	}
	return mapReport(rep), nil
}

func (r *repo) Get(ctx context.Context, id string) (Report, error) {
	rep, err := r.q.GetReport(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrNotFound
	}
	if err != nil {
		return Report{}, err
	}
	return mapReport(rep), nil
}

func (r *repo) List(ctx context.Context, status string) ([]Report, error) {
	if status == "" {
		rows, err := r.q.ListAllReports(ctx)
		return mapReports(rows), err
	}
	rows, err := r.q.ListReports(ctx, status)
	return mapReports(rows), err
}

func (r *repo) Resolve(ctx context.Context, id, status, resolution, handledBy string) error {
	return r.q.ResolveReport(ctx, sqlc.ResolveReportParams{
		ID: id, Status: status, Resolution: resolution, HandledBy: &handledBy,
	})
}
