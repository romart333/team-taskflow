package comment_list

import (
	"context"
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
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{}, &commentRepoMock{comments: expected})

		out, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Comments)
	})

	t.Run("task not found", func(t *testing.T) {
		uc := New(&taskRepoMock{err: domain.ErrNotFound}, &teamRepoMock{}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		uc := New(&taskRepoMock{task: task}, &teamRepoMock{err: domain.ErrNotFound}, &commentRepoMock{})

		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}
