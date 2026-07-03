package domain

import "time"

type TaskComment struct {
	ID        int64
	TaskID    int64
	UserID    int64
	Body      string
	CreatedAt time.Time
}

// ValidateCommentBody checks the invariants for a new comment.
func ValidateCommentBody(body string) error {
	if body == "" {
		return NewValidationError("comment body is required")
	}
	return nil
}
