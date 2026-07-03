package task_update

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type taskRepoMock struct {
	task      domain.Task
	getErr    error
	updateErr error
	gotUpdate *domain.Task
}

func (m *taskRepoMock) GetByID(context.Context, int64) (domain.Task, error) {
	if m.gotUpdate != nil {
		return *m.gotUpdate, m.getErr
	}
	return m.task, m.getErr
}

func (m *taskRepoMock) Update(_ context.Context, task domain.Task) error {
	if m.updateErr == nil {
		m.gotUpdate = &task
	}
	return m.updateErr
}

type teamRepoMock struct {
	errs map[int64]error
}

func (m *teamRepoMock) GetMember(_ context.Context, _ int64, userID int64) (domain.TeamMember, error) {
	if err, ok := m.errs[userID]; ok {
		return domain.TeamMember{}, err
	}
	return domain.TeamMember{UserID: userID, Role: domain.RoleMember}, nil
}

type historyRepoMock struct {
	entries []domain.TaskHistoryEntry
	err     error
}

func (m *historyRepoMock) AddEntries(_ context.Context, entries []domain.TaskHistoryEntry) error {
	m.entries = append(m.entries, entries...)
	return m.err
}

type txMock struct{ calls int }

func (m *txMock) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type invalidatorMock struct{ calls int }

func (m *invalidatorMock) InvalidateTeam(context.Context, int64) error {
	m.calls++
	return nil
}

func TestUsecase_Handle(t *testing.T) {
	baseTask := domain.Task{
		ID: 7, TeamID: 1, Title: "Old title", Description: "Old desc",
		Status: domain.TaskStatusTodo, CreatedBy: 5,
	}

	t.Run("changes are persisted with history in one transaction", func(t *testing.T) {
		tasks := &taskRepoMock{task: baseTask}
		history := &historyRepoMock{}
		transaction := &txMock{}
		cache := &invalidatorMock{}
		uc := New(tasks, &teamRepoMock{}, history, transaction, cache)

		out, err := uc.Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7,
			Title:  new("New title"),
			Status: new("done"),
		})

		require.NoError(t, err)
		assert.Equal(t, "New title", out.Task.Title)
		assert.Equal(t, domain.TaskStatusDone, out.Task.Status)
		assert.Equal(t, 1, transaction.calls)
		assert.Equal(t, 1, cache.calls)
		require.Len(t, history.entries, 2)
		assert.Equal(t, domain.TaskHistoryEntry{
			TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldTitle,
			OldValue: "Old title", NewValue: "New title",
		}, history.entries[0])
		assert.Equal(t, domain.TaskHistoryEntry{
			TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldStatus,
			OldValue: "todo", NewValue: "done",
		}, history.entries[1])
	})

	t.Run("no-op update skips transaction and history", func(t *testing.T) {
		tasks := &taskRepoMock{task: baseTask}
		history := &historyRepoMock{}
		transaction := &txMock{}
		cache := &invalidatorMock{}
		uc := New(tasks, &teamRepoMock{}, history, transaction, cache)

		out, err := uc.Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("Old title"),
		})

		require.NoError(t, err)
		assert.Equal(t, baseTask, out.Task)
		assert.Zero(t, transaction.calls)
		assert.Empty(t, history.entries)
		assert.Zero(t, cache.calls)
	})

	t.Run("task not found", func(t *testing.T) {
		tasks := &taskRepoMock{getErr: domain.ErrNotFound}
		uc := New(tasks, &teamRepoMock{}, &historyRepoMock{}, &txMock{}, &invalidatorMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		tasks := &taskRepoMock{task: baseTask}
		teams := &teamRepoMock{errs: map[int64]error{5: domain.ErrNotFound}}
		uc := New(tasks, teams, &historyRepoMock{}, &txMock{}, &invalidatorMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Title: new("x")})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("invalid status", func(t *testing.T) {
		tasks := &taskRepoMock{task: baseTask}
		uc := New(tasks, &teamRepoMock{}, &historyRepoMock{}, &txMock{}, &invalidatorMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Status: new("archived")})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("assignee outside the team", func(t *testing.T) {
		outsider := int64(99)
		tasks := &taskRepoMock{task: baseTask}
		teams := &teamRepoMock{errs: map[int64]error{99: domain.ErrNotFound}}
		uc := New(tasks, teams, &historyRepoMock{}, &txMock{}, &invalidatorMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, AssigneeID: &outsider})

		require.ErrorIs(t, err, domain.ErrValidation)
	})
}
