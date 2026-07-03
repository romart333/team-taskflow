package team_create

import (
	"context"
	"fmt"
	"log/slog"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	teams TeamRepository
	tx    TxManager
}

func New(teams TeamRepository, tx TxManager) *Usecase {
	return &Usecase{teams: teams, tx: tx}
}

// Handle creates a team and makes the actor its owner atomically.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if err := domain.ValidateTeamName(in.Name); err != nil {
		return Output{}, fmt.Errorf("validating team: %w", err)
	}

	var teamID int64
	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		id, err := u.teams.CreateTeam(txCtx, domain.Team{Name: in.Name, CreatedBy: in.ActorID})
		if err != nil {
			return fmt.Errorf("creating team: %w", err)
		}

		if err := u.teams.AddMember(txCtx, domain.TeamMember{
			TeamID: id,
			UserID: in.ActorID,
			Role:   domain.RoleOwner,
		}); err != nil {
			return fmt.Errorf("adding owner membership: %w", err)
		}

		teamID = id
		return nil
	})
	if err != nil {
		return Output{}, fmt.Errorf("creating team transactionally: %w", err)
	}

	slog.InfoContext(ctx, "team created", "team_id", teamID, "owner_id", in.ActorID)
	return Output{TeamID: teamID, Name: in.Name}, nil
}
