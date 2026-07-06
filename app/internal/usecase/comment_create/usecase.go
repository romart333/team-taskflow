package comment_create

import (
	"context"
	"fmt"
	"log/slog"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	access   TaskAccess
	comments CommentRepository
}

func New(access TaskAccess, comments CommentRepository) *Usecase {
	return &Usecase{access: access, comments: comments}
}

// Handle adds a comment to a task on behalf of a team member.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if err := domain.ValidateCommentBody(in.Body); err != nil {
		return Output{}, fmt.Errorf("validating comment: %w", err)
	}

	if _, err := u.access.LoadTaskForMember(ctx, in.TaskID, in.ActorID); err != nil {
		return Output{}, fmt.Errorf("authorizing task access: %w", err)
	}

	commentID, err := u.comments.Create(ctx, domain.TaskComment{
		TaskID: in.TaskID,
		UserID: in.ActorID,
		Body:   in.Body,
	})
	if err != nil {
		return Output{}, fmt.Errorf("creating comment: %w", err)
	}

	comment, err := u.comments.GetByID(ctx, commentID)
	if err != nil {
		return Output{}, fmt.Errorf("loading created comment: %w", err)
	}

	slog.InfoContext(ctx, "comment added", "task_id", in.TaskID, "comment_id", commentID, "user_id", in.ActorID)
	return Output{Comment: comment}, nil
}
