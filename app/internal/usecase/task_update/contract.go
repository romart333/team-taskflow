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
	// SetAssignee marks an explicit assignee change (a JSON null is
	// indistinguishable from an absent field otherwise): AssigneeID nil then
	// unassigns the task, non-nil assigns the given member.
	SetAssignee bool
	AssigneeID  *int64
}

type Output struct {
	Task domain.Task
}

// TaskRepository is the persistence port for tasks.
type TaskRepository interface {
	// GetByID returns domain.ErrNotFound when the task does not exist.
	GetByID(ctx context.Context, taskID int64) (domain.Task, error)
	// GetByIDForUpdate additionally locks the row until the surrounding
	// transaction ends, serializing concurrent updates of the same task so
	// they cannot overwrite each other's fields from stale snapshots.
	GetByIDForUpdate(ctx context.Context, taskID int64) (domain.Task, error)
	Update(ctx context.Context, task domain.Task) error
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
