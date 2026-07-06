package task_create

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	tasks  TaskRepository
	access TeamAccess
	teams  TeamRepository
	cache  TaskCacheInvalidator
}

func New(tasks TaskRepository, access TeamAccess, teams TeamRepository, cache TaskCacheInvalidator) *Usecase {
	return &Usecase{tasks: tasks, access: access, teams: teams, cache: cache}
}

// Handle creates a task in a team. Both the author and the assignee (when
// set) must be members of the team.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if err := domain.ValidateNewTask(in.Title); err != nil {
		return Output{}, fmt.Errorf("validating task: %w", err)
	}

	if err := u.access.EnsureTeamMember(ctx, in.TeamID, in.ActorID); err != nil {
		return Output{}, fmt.Errorf("authorizing author: %w", err)
	}

	if in.AssigneeID != nil && *in.AssigneeID != in.ActorID {
		if _, err := u.teams.GetMember(ctx, in.TeamID, *in.AssigneeID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return Output{}, fmt.Errorf("checking assignee membership: %w",
					domain.NewValidationError("assignee is not a member of this team"))
			}
			return Output{}, fmt.Errorf("getting assignee membership: %w", err)
		}
	}

	taskID, err := u.tasks.Create(ctx, domain.Task{
		TeamID:      in.TeamID,
		Title:       in.Title,
		Description: in.Description,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  in.AssigneeID,
		CreatedBy:   in.ActorID,
	})
	if err != nil {
		return Output{}, fmt.Errorf("creating task: %w", err)
	}

	task, err := u.tasks.GetByID(ctx, taskID)
	if err != nil {
		return Output{}, fmt.Errorf("loading created task: %w", err)
	}

	if err := u.cache.InvalidateTeam(ctx, in.TeamID); err != nil {
		slog.WarnContext(ctx, "task list cache invalidation failed", "team_id", in.TeamID, "error", err)
	}

	slog.InfoContext(ctx, "task created", "task_id", taskID, "team_id", in.TeamID, "created_by", in.ActorID)
	return Output{Task: task}, nil
}
