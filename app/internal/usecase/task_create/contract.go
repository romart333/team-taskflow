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

// TeamAccess authorizes team memberships of the actor and the assignee.
type TeamAccess interface {
	// EnsureTeamMember returns a client-visible domain.ErrPermissionDenied
	// when the actor is not a member of the team.
	EnsureTeamMember(ctx context.Context, teamID, actorID int64) error
	// EnsureAssigneeMember returns a client-visible domain.ErrValidation
	// when the assignee is not a member of the team.
	EnsureAssigneeMember(ctx context.Context, teamID, assigneeID int64) error
}

// TaskCacheInvalidator drops cached task listings of a team (best effort).
type TaskCacheInvalidator interface {
	InvalidateTeam(ctx context.Context, teamID int64) error
}
