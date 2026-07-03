package team_create

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	Name    string
}

type Output struct {
	TeamID int64
	Name   string
}

// TeamRepository is the persistence port for teams and memberships.
type TeamRepository interface {
	CreateTeam(ctx context.Context, team domain.Team) (int64, error)
	AddMember(ctx context.Context, member domain.TeamMember) error
}

// TxManager controls the transaction boundary of the operation.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
