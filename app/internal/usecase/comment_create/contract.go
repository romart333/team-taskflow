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

// TaskAccess loads a task and authorizes the actor as a member of its team.
type TaskAccess interface {
	// LoadTaskForMember returns a client-visible domain.ErrNotFound when the
	// task is missing and domain.ErrPermissionDenied when the actor is not a
	// member of the task's team.
	LoadTaskForMember(ctx context.Context, taskID, actorID int64) (domain.Task, error)
}

// CommentRepository is the persistence port for task comments.
type CommentRepository interface {
	Create(ctx context.Context, comment domain.TaskComment) (int64, error)
	// GetByID returns domain.ErrNotFound when the comment does not exist.
	GetByID(ctx context.Context, commentID int64) (domain.TaskComment, error)
}
