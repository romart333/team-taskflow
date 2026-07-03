package comment_list

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	TaskID  int64
}

type Output struct {
	Comments []domain.TaskComment
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

// CommentRepository is the read port for task comments.
type CommentRepository interface {
	ListByTask(ctx context.Context, taskID int64) ([]domain.TaskComment, error)
}
