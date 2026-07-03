package taskrepo

import (
	"database/sql"
	"time"

	"team-taskflow/internal/domain"
)

type taskEntity struct {
	ID          int64         `db:"id"`
	TeamID      int64         `db:"team_id"`
	Title       string        `db:"title"`
	Description string        `db:"description"`
	Status      string        `db:"status"`
	AssigneeID  sql.NullInt64 `db:"assignee_id"`
	CreatedBy   int64         `db:"created_by"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

func (e taskEntity) toDomain() domain.Task {
	task := domain.Task{
		ID:          e.ID,
		TeamID:      e.TeamID,
		Title:       e.Title,
		Description: e.Description,
		Status:      domain.TaskStatus(e.Status),
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if e.AssigneeID.Valid {
		task.AssigneeID = &e.AssigneeID.Int64
	}
	return task
}

func assigneeParam(id *int64) sql.NullInt64 {
	if id == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *id, Valid: true}
}
