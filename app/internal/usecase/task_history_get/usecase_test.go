package task_history_get

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

func TestUsecase_Handle(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}

	t.Run("success", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		history := NewMockHistoryRepository(t)
		expected := []domain.TaskHistoryEntry{{ID: 1, TaskID: 7, Field: domain.TaskFieldStatus}}
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		history.EXPECT().ListByTask(mock.Anything, int64(7)).Return(expected, nil)

		uc := New(access, history)
		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Entries)
	})

	t.Run("task not found", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		history := NewMockHistoryRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewNotFoundError("task not found"))

		uc := New(access, history)
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		history := NewMockHistoryRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewPermissionDeniedError("not a member"))

		uc := New(access, history)
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}
	dbErr := errors.New("db down")

	t.Run("access check failure", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		history := NewMockHistoryRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(domain.Task{}, dbErr)

		uc := New(access, history)
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("history load failure", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		history := NewMockHistoryRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		history.EXPECT().ListByTask(mock.Anything, int64(7)).Return(nil, dbErr)

		uc := New(access, history)
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
	})
}
