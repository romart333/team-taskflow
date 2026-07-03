package task_history_get

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	TaskID  int64
}

type Output struct {
	Entries []domain.TaskHistoryEntry
}

// TaskRepository is the read port for tasks.
type TaskRepository interface {
	// GetByID returns domain.ErrNotFound when the task does not exist.
	GetByID(ctx context.Context, taskID int64) (domain.Task, error)
}

// TeamRepository checks team memberships.
type TeamRepository interface {
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
}

// HistoryRepository is the read port for task audit entries.
type HistoryRepository interface {
	ListByTask(ctx context.Context, taskID int64) ([]domain.TaskHistoryEntry, error)
}
