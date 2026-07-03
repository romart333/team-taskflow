package comment_create

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	TaskID  int64
	Body    string
}

type Output struct {
	Comment domain.TaskComment
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

// CommentRepository is the persistence port for task comments.
type CommentRepository interface {
	Create(ctx context.Context, comment domain.TaskComment) (int64, error)
	// GetByID returns domain.ErrNotFound when the comment does not exist.
	GetByID(ctx context.Context, commentID int64) (domain.TaskComment, error)
}
