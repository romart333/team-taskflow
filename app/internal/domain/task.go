package domain

import (
	"strconv"
	"time"
)

// TaskStatus is the workflow state of a task.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// ParseTaskStatus validates a raw status string coming from the outside.
func ParseTaskStatus(raw string) (TaskStatus, error) {
	switch TaskStatus(raw) {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return TaskStatus(raw), nil
	default:
		return "", NewValidationError("status must be one of: todo, in_progress, done")
	}
}

// Audited task field names, recorded in task history entries.
const (
	TaskFieldTitle       = "title"
	TaskFieldDescription = "description"
	TaskFieldStatus      = "status"
	TaskFieldAssignee    = "assignee_id"
)

type Task struct {
	ID          int64
	TeamID      int64
	Title       string
	Description string
	Status      TaskStatus
	AssigneeID  *int64
	CreatedBy   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ValidateNewTask checks the invariants for a new task.
func ValidateNewTask(title string) error {
	if title == "" {
		return NewValidationError("task title is required")
	}
	return nil
}

// FieldChange is a single audited change of a task field.
type FieldChange struct {
	Field    string
	OldValue string
	NewValue string
}

// Diff returns the audited field changes between the task and its updated version.
func (t Task) Diff(updated Task) []FieldChange {
	var changes []FieldChange
	if t.Title != updated.Title {
		changes = append(changes, FieldChange{TaskFieldTitle, t.Title, updated.Title})
	}
	if t.Description != updated.Description {
		changes = append(changes, FieldChange{TaskFieldDescription, t.Description, updated.Description})
	}
	if t.Status != updated.Status {
		changes = append(changes, FieldChange{TaskFieldStatus, string(t.Status), string(updated.Status)})
	}
	if formatAssignee(t.AssigneeID) != formatAssignee(updated.AssigneeID) {
		changes = append(changes, FieldChange{TaskFieldAssignee, formatAssignee(t.AssigneeID), formatAssignee(updated.AssigneeID)})
	}
	return changes
}

func formatAssignee(id *int64) string {
	if id == nil {
		return ""
	}
	return strconv.FormatInt(*id, 10)
}
