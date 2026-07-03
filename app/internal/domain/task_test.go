package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Diff(t *testing.T) {
	assignee := int64(7)
	otherAssignee := int64(9)

	tests := []struct {
		name     string
		current  Task
		updated  Task
		expected []FieldChange
	}{
		{
			name:     "no changes",
			current:  Task{Title: "a", Description: "d", Status: TaskStatusTodo, AssigneeID: &assignee},
			updated:  Task{Title: "a", Description: "d", Status: TaskStatusTodo, AssigneeID: &assignee},
			expected: nil,
		},
		{
			name:    "title and status changed",
			current: Task{Title: "a", Status: TaskStatusTodo},
			updated: Task{Title: "b", Status: TaskStatusDone},
			expected: []FieldChange{
				{Field: TaskFieldTitle, OldValue: "a", NewValue: "b"},
				{Field: TaskFieldStatus, OldValue: "todo", NewValue: "done"},
			},
		},
		{
			name:    "assignee set from nil",
			current: Task{Title: "a"},
			updated: Task{Title: "a", AssigneeID: &assignee},
			expected: []FieldChange{
				{Field: TaskFieldAssignee, OldValue: "", NewValue: "7"},
			},
		},
		{
			name:    "assignee replaced",
			current: Task{Title: "a", AssigneeID: &assignee},
			updated: Task{Title: "a", AssigneeID: &otherAssignee},
			expected: []FieldChange{
				{Field: TaskFieldAssignee, OldValue: "7", NewValue: "9"},
			},
		},
		{
			name:    "description changed",
			current: Task{Description: "old"},
			updated: Task{Description: "new"},
			expected: []FieldChange{
				{Field: TaskFieldDescription, OldValue: "old", NewValue: "new"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.current.Diff(tt.updated))
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	for _, valid := range []string{"todo", "in_progress", "done"} {
		status, err := ParseTaskStatus(valid)
		require.NoError(t, err)
		assert.Equal(t, TaskStatus(valid), status)
	}

	_, err := ParseTaskStatus("archived")
	require.ErrorIs(t, err, ErrValidation)
}

func TestValidateNewTask(t *testing.T) {
	assert.NoError(t, ValidateNewTask("title"))
	assert.ErrorIs(t, ValidateNewTask(""), ErrValidation)
}
