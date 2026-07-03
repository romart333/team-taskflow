package task_create

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID     int64
	TeamID      int64
	Title       string
	Description string
	AssigneeID  *int64
}

type Output struct {
	Task domain.Task
}

// TaskRepository is the persistence port for tasks.
type TaskRepository interface {
	Create(ctx context.Context, task domain.Task) (int64, error)
	// GetByID returns domain.ErrNotFound when the task does not exist.
	GetByID(ctx context.Context, taskID int64) (domain.Task, error)
}

// TeamRepository checks team memberships.
type TeamRepository interface {
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
}

// TaskCacheInvalidator drops cached task listings of a team (best effort).
type TaskCacheInvalidator interface {
	InvalidateTeam(ctx context.Context, teamID int64) error
}
