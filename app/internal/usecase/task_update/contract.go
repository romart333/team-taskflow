package task_update

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	TaskID  int64
	// nil pointers mean "leave the field unchanged".
	Title       *string
	Description *string
	Status      *string
	AssigneeID  *int64
}

type Output struct {
	Task domain.Task
}

// TaskRepository is the persistence port for tasks.
type TaskRepository interface {
	// GetByID returns domain.ErrNotFound when the task does not exist.
	GetByID(ctx context.Context, taskID int64) (domain.Task, error)
	Update(ctx context.Context, task domain.Task) error
}

// TeamRepository checks team memberships.
type TeamRepository interface {
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
}

// HistoryRepository persists task audit entries.
type HistoryRepository interface {
	AddEntries(ctx context.Context, entries []domain.TaskHistoryEntry) error
}

// TxManager controls the transaction boundary of the operation.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// TaskCacheInvalidator drops cached task listings of a team (best effort).
type TaskCacheInvalidator interface {
	InvalidateTeam(ctx context.Context, teamID int64) error
}
