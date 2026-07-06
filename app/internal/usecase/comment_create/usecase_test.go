package comment_create

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
	createID   int64
	createErr  error
	comment    domain.TaskComment
	gotComment domain.TaskComment
}

func (m *commentRepoMock) Create(_ context.Context, comment domain.TaskComment) (int64, error) {
	m.gotComment = comment
	return m.createID, m.createErr
}

func (m *commentRepoMock) GetByID(context.Context, int64) (domain.TaskComment, error) {
	return m.comment, nil
}

func TestUsecase_Handle(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}

	t.Run("success", func(t *testing.T) {
		comments := &commentRepoMock{createID: 3, comment: domain.TaskComment{ID: 3, TaskID: 7, Body: "LGTM"}}
		uc := New(&accessMock{task: task}, comments)

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "LGTM"})

		require.NoError(t, err)
		assert.Equal(t, int64(3), out.Comment.ID)
		assert.Equal(t, domain.TaskComment{TaskID: 7, UserID: 5, Body: "LGTM"}, comments.gotComment)
	})

	t.Run("empty body", func(t *testing.T) {
		uc := New(&accessMock{task: task}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: ""})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("task not found", func(t *testing.T) {
		uc := New(&accessMock{err: domain.NewNotFoundError("task not found")}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		uc := New(&accessMock{err: domain.NewPermissionDeniedError("not a member")}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 1}
	dbErr := errors.New("db down")

	t.Run("access check failure", func(t *testing.T) {
		uc := New(&accessMock{err: dbErr}, &commentRepoMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrNotFound)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("create failure", func(t *testing.T) {
		uc := New(&accessMock{task: task}, &commentRepoMock{createErr: dbErr})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Body: "hi"})
		require.Error(t, err)
	})
}
