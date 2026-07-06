package taskaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type taskGetterMock struct {
	task domain.Task
	err  error
}

func (m *taskGetterMock) GetByID(context.Context, int64) (domain.Task, error) {
	return m.task, m.err
}

type memberGetterMock struct {
	err error
}

func (m *memberGetterMock) GetMember(context.Context, int64, int64) (domain.TeamMember, error) {
	if m.err != nil {
		return domain.TeamMember{}, m.err
	}
	return domain.TeamMember{Role: domain.RoleMember}, nil
}

func TestService(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 3, Title: "t"}
	dbErr := errors.New("db down")

	t.Run("member gets the task", func(t *testing.T) {
		svc := New(&taskGetterMock{task: task}, &memberGetterMock{})

		got, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.NoError(t, err)
		assert.Equal(t, task, got)
	})

	t.Run("missing task maps to client-visible not found", func(t *testing.T) {
		svc := New(&taskGetterMock{err: domain.ErrNotFound}, &memberGetterMock{})

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, domain.ErrNotFound)
		var safeErr *domain.SafeError
		require.ErrorAs(t, err, &safeErr)
		assert.Equal(t, "task not found", safeErr.Msg)
	})

	t.Run("non-member is rejected with permission denied", func(t *testing.T) {
		svc := New(&taskGetterMock{task: task}, &memberGetterMock{err: domain.ErrNotFound})

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("task repo failure is passed through", func(t *testing.T) {
		svc := New(&taskGetterMock{err: dbErr}, &memberGetterMock{})

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, dbErr)
		require.NotErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("membership repo failure is not permission denied", func(t *testing.T) {
		svc := New(&taskGetterMock{task: task}, &memberGetterMock{err: dbErr})

		err := svc.EnsureTeamMember(context.Background(), 3, 5)

		require.ErrorIs(t, err, dbErr)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("member passes the team check", func(t *testing.T) {
		svc := New(&taskGetterMock{}, &memberGetterMock{})

		require.NoError(t, svc.EnsureTeamMember(context.Background(), 3, 5))
	})
}
