package comment_create

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
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		comments.EXPECT().Create(mock.Anything, mock.MatchedBy(func(comment domain.TaskComment) bool {
			return comment == domain.TaskComment{TaskID: 7, UserID: 5, Body: "LGTM"}
		})).Return(3, nil)
		comments.EXPECT().GetByID(mock.Anything, int64(3)).Return(domain.TaskComment{ID: 3, TaskID: 7, Body: "LGTM"}, nil)
		uc := New(access, comments)

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "LGTM"})

		require.NoError(t, err)
		assert.Equal(t, int64(3), out.Comment.ID)
	})

	t.Run("empty body", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: ""})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("task not found", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewNotFoundError("task not found"))
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).
			Return(domain.Task{}, domain.NewPermissionDeniedError("not a member"))
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})

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

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("create failure", func(t *testing.T) {
		access := NewMockTaskAccess(t)
		comments := NewMockCommentRepository(t)
		access.EXPECT().LoadTaskForMember(mock.Anything, int64(7), int64(5)).Return(task, nil)
		comments.EXPECT().Create(mock.Anything, mock.Anything).Return(0, dbErr)
		uc := New(access, comments)

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})
		require.Error(t, err)
	})
}
