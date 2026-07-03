// Package analytics_get serves the reporting reads: team stats, per-team top
// task creators and the assignee integrity audit. The three reads share one
// responsibility (analytics reporting) and one repository port, so they live
// in a single usecase package.
package analytics_get

import (
	"context"

	"team-taskflow/internal/domain"
)

// AnalyticsRepository is the read port for reporting queries.
type AnalyticsRepository interface {
	TeamStats(ctx context.Context, doneWindowDays int) ([]domain.TeamStats, error)
	TopCreators(ctx context.Context, windowDays, limit int) ([]domain.TeamTopCreator, error)
	OrphanedAssignees(ctx context.Context) ([]domain.OrphanedAssigneeTask, error)
}
