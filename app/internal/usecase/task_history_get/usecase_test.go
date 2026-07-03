package task_history_get

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type taskRepoMock struct {
	task domain.Task
	err  error
}

func (m *taskRepoMock) GetByID(context.Context, int64) (domain.Task, error) {
	return m.task, m.err
}

type teamRepoMock struct{ err error }

func (m *teamRepoMock) GetMember(context.Context, int64, int64) (domain.TeamMember, error) {
	if m.err != nil {
		return domain.TeamMember{}, m.err
	}
	return domain.TeamMember{Role: domain.RoleMember}, nil
}

type historyRepoMock struct {
	entries []domain.TaskHistoryEntry
	err     error
}

func (m *historyRepoMock) ListByTask(context.Context, int64) ([]domain.TaskHistoryEntry, error) {
	return m.entries, m.err
}

func TestUsecase_Handle(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}

	t.Run("success", func(t *testing.T) {
		expected := []domain.TaskHistoryEntry{{ID: 1, TaskID: 7, Field: domain.TaskFieldStatus}}
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{}, &historyRepoMock{entries: expected})

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Entries)
	})

	t.Run("task not found", func(t *testing.T) {
		uc := New(&taskRepoMock{err: domain.ErrNotFound}, &teamRepoMock{}, &historyRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{err: domain.ErrNotFound}, &historyRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}
	dbErr := errors.New("db down")

	t.Run("task load failure", func(t *testing.T) {
		uc := New(&taskRepoMock{err: dbErr}, &teamRepoMock{}, &historyRepoMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("membership load failure", func(t *testing.T) {
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{err: dbErr}, &historyRepoMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("history load failure", func(t *testing.T) {
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{}, &historyRepoMock{err: dbErr})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
	})
}
