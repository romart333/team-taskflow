package comment_list

import (
	"context"
	"errors"
	"fmt"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	tasks    TaskRepository
	teams    TeamRepository
	comments CommentRepository
}

func New(tasks TaskRepository, teams TeamRepository, comments CommentRepository) *Usecase {
	return &Usecase{tasks: tasks, teams: teams, comments: comments}
}

// Handle returns a task's comments, visible to team members only.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	task, err := u.tasks.GetByID(ctx, in.TaskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("loading task: %w", domain.NewNotFoundError("task not found"))
		}
		return Output{}, fmt.Errorf("loading task: %w", err)
	}

	if _, err := u.teams.GetMember(ctx, task.TeamID, in.ActorID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("checking membership: %w",
				domain.NewPermissionDeniedError("you are not a member of this task's team"))
		}
		return Output{}, fmt.Errorf("getting membership: %w", err)
	}

	comments, err := u.comments.ListByTask(ctx, in.TaskID)
	if err != nil {
		return Output{}, fmt.Errorf("listing comments: %w", err)
	}
	return Output{Comments: comments}, nil
}
