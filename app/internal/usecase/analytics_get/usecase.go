package analytics_get

import (
	"context"
	"fmt"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	analytics AnalyticsRepository
}

func New(analytics AnalyticsRepository) *Usecase {
	return &Usecase{analytics: analytics}
}

// TeamStats reports member count and recently done tasks for the actor's teams.
func (u *Usecase) TeamStats(ctx context.Context, actorID int64) ([]domain.TeamStats, error) {
	stats, err := u.analytics.TeamStats(ctx, actorID, domain.TeamStatsDoneWindowDays)
	if err != nil {
		return nil, fmt.Errorf("loading team stats: %w", err)
	}
	return stats, nil
}

// TopCreators reports the top task creators over the last month per team the
// actor belongs to.
func (u *Usecase) TopCreators(ctx context.Context, actorID int64) ([]domain.TeamTopCreator, error) {
	creators, err := u.analytics.TopCreators(ctx, actorID, domain.TopCreatorsWindowDays, domain.TopCreatorsLimit)
	if err != nil {
		return nil, fmt.Errorf("loading top creators: %w", err)
	}
	return creators, nil
}

// OrphanedAssignees reports tasks in the actor's teams whose assignee left
// (or never was in) the task's team — a data integrity audit.
func (u *Usecase) OrphanedAssignees(ctx context.Context, actorID int64) ([]domain.OrphanedAssigneeTask, error) {
	tasks, err := u.analytics.OrphanedAssignees(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("loading orphaned assignees: %w", err)
	}
	return tasks, nil
}
