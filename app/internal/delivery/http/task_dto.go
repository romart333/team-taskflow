package http

import (
	"time"

	"team-taskflow/internal/domain"
)

// taskResponse is the transport representation of a task.
type taskResponse struct {
	ID          int64     `json:"id"`
	TeamID      int64     `json:"team_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	AssigneeID  *int64    `json:"assignee_id,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toTaskResponse(task domain.Task) taskResponse {
	return taskResponse{
		ID:          task.ID,
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		AssigneeID:  task.AssigneeID,
		CreatedBy:   task.CreatedBy,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}
