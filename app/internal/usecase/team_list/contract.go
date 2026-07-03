package team_list

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
}

type Output struct {
	Teams []domain.TeamWithRole
}

// TeamRepository is the read port for team memberships.
type TeamRepository interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.TeamWithRole, error)
}
