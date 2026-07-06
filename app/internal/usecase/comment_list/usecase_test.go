package comment_list

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type accessMock struct {
	task domain.Task
	err  error
}

func (m *accessMock) LoadTaskForMember(context.Context, int64, int64) (domain.Task, error) {
	return m.task, m.err
}

type commentRepoMock struct {
	comments []domain.TaskComment
	err      error
}

func (m *commentRepoMock) ListByTask(context.Context, int64) ([]domain.TaskComment, error) {
	return m.comments, m.err
}

func TestUsecase_Handle(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}

	t.Run("success", func(t *testing.T) {
		expected := []domain.TaskComment{{ID: 1, TaskID: 7, Body: "hi"}}
		uc := New(&accessMock{task: task}, &commentRepoMock{comments: expected})

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Comments)
	})

	t.Run("task not found", func(t *testing.T) {
		uc := New(&accessMock{err: domain.NewNotFoundError("task not found")}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		uc := New(&accessMock{err: domain.NewPermissionDeniedError("not a member")}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}
	dbErr := errors.New("db down")

	t.Run("access check failure", func(t *testing.T) {
		uc := New(&accessMock{err: dbErr}, &commentRepoMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("list failure", func(t *testing.T) {
		uc := New(&accessMock{task: task}, &commentRepoMock{err: dbErr})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})
		require.Error(t, err)
	})
}
