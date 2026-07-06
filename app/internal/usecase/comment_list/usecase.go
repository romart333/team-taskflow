package comment_list

import (
	"context"
	"fmt"
)

type Usecase struct {
	access   TaskAccess
	comments CommentRepository
}

func New(access TaskAccess, comments CommentRepository) *Usecase {
	return &Usecase{access: access, comments: comments}
}

// Handle returns a task's comments, visible to team members only.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if _, err := u.access.LoadTaskForMember(ctx, in.TaskID, in.ActorID); err != nil {
		return Output{}, fmt.Errorf("authorizing task access: %w", err)
	}

	comments, err := u.comments.ListByTask(ctx, in.TaskID)
	if err != nil {
		return Output{}, fmt.Errorf("listing comments: %w", err)
	}
	return Output{Comments: comments}, nil
}
