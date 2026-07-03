package commentrepo

import (
	"time"

	"team-taskflow/internal/domain"
)

type commentEntity struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	UserID    int64     `db:"user_id"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
}

func (e commentEntity) toDomain() domain.TaskComment {
	return domain.TaskComment{
		ID:        e.ID,
		TaskID:    e.TaskID,
		UserID:    e.UserID,
		Body:      e.Body,
		CreatedAt: e.CreatedAt,
	}
}
