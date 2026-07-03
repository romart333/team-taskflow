package comment_create

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

// Handle adds a comment to a task on behalf of a team member.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if err := domain.ValidateCommentBody(in.Body); err != nil {
		return Output{}, fmt.Errorf("validating comment: %w", err)
	}

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

	commentID, err := u.comments.Create(ctx, domain.TaskComment{
		TaskID: in.TaskID,
		UserID: in.ActorID,
		Body:   in.Body,
	})
	if err != nil {
		return Output{}, fmt.Errorf("creating comment: %w", err)
	}

	comment, err := u.comments.GetByID(ctx, commentID)
	if err != nil {
		return Output{}, fmt.Errorf("loading created comment: %w", err)
	}

	slog.InfoContext(ctx, "comment added", "task_id", in.TaskID, "comment_id", commentID, "user_id", in.ActorID)
	return Output{Comment: comment}, nil
}
