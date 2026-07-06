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

// TaskAccess loads a task and authorizes the actor as a member of its team.
type TaskAccess interface {
	// LoadTaskForMember returns a client-visible domain.ErrNotFound when the
	// task is missing and domain.ErrPermissionDenied when the actor is not a
	// member of the task's team.
	LoadTaskForMember(ctx context.Context, taskID, actorID int64) (domain.Task, error)
}

// CommentRepository is the read port for task comments.
type CommentRepository interface {
	ListByTask(ctx context.Context, taskID int64) ([]domain.TaskComment, error)
}
