package comment_list

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
		comments := NewMockCommentRepository(t)
		expected := []domain.TaskComment{{ID: 1, TaskID: 7, Body: "hi"}}
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		comments.EXPECT().ListByTask(mock.Anything, int64(7)).Return(expected, nil)
		uc := New(access, comments)

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Comments)
	})

	t.Run("task not found", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewNotFoundError("task not found"))
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewPermissionDeniedError("not a member"))
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}
	dbErr := errors.New("db down")

	t.Run("access check failure", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(domain.Task{}, dbErr)
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("list failure", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		comments.EXPECT().ListByTask(mock.Anything, int64(7)).Return(nil, dbErr)
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
	})
}
