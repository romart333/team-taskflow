package team_list

import (
	"context"
	"fmt"
)

type Usecase struct {
	teams TeamRepository
}

func New(teams TeamRepository) *Usecase {
	return &Usecase{teams: teams}
}

// Handle returns the teams the actor belongs to, with the actor's role in each.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	teams, err := u.teams.ListByUser(ctx, in.ActorID)
	if err != nil {
		return Output{}, fmt.Errorf("listing teams for user: %w", err)
	}
	return Output{Teams: teams}, nil
}
