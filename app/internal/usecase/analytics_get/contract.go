// Package analytics_get serves the reporting reads: team stats, per-team top
// task creators and the assignee integrity audit. The three reads share one
// responsibility (analytics reporting) and one repository port, so they live
// in a single usecase package.
package analytics_get

import (
	"context"

	"team-taskflow/internal/domain"
)

// AnalyticsRepository is the read port for reporting queries. Every query is
// scoped to teams the actor is a member of: analytics must never leak other
// teams' data to an arbitrary authenticated user.
type AnalyticsRepository interface {
	TeamStats(ctx context.Context, actorID int64, doneWindowDays int) ([]domain.TeamStats, error)
	TopCreators(ctx context.Context, actorID int64, windowDays, limit int) ([]domain.TeamTopCreator, error)
	OrphanedAssignees(ctx context.Context, actorID int64) ([]domain.OrphanedAssigneeTask, error)
}
