package taskaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

func TestService(t *testing.T) {
	task := domain.Task{ID: 7, TeamID: 3, Title: "t"}
	dbErr := errors.New("db down")

	t.Run("member gets the task", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).Return(task, nil)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(5)).Return(domain.TeamMember{Role: domain.RoleMember}, nil)
		svc := New(tasks, members)

		got, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.NoError(t, err)
		assert.Equal(t, task, got)
	})

	t.Run("missing task maps to client-visible not found", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).Return(domain.Task{}, domain.ErrNotFound)
		svc := New(tasks, members)

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, domain.ErrNotFound)
		var safeErr *domain.SafeError
		require.ErrorAs(t, err, &safeErr)
		assert.Equal(t, "task not found", safeErr.Msg)
	})

	t.Run("non-member is rejected with permission denied", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).Return(task, nil)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(5)).Return(domain.TeamMember{}, domain.ErrNotFound)
		svc := New(tasks, members)

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("task repo failure is passed through", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).Return(domain.Task{}, dbErr)
		svc := New(tasks, members)

		_, err := svc.LoadTaskForMember(context.Background(), 7, 5)

		require.ErrorIs(t, err, dbErr)
		require.NotErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("membership repo failure is not permission denied", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(5)).Return(domain.TeamMember{}, dbErr)
		svc := New(tasks, members)

		err := svc.EnsureTeamMember(context.Background(), 3, 5)

		require.ErrorIs(t, err, dbErr)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("member passes the team check", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(5)).Return(domain.TeamMember{Role: domain.RoleMember}, nil)
		svc := New(tasks, members)

		require.NoError(t, svc.EnsureTeamMember(context.Background(), 3, 5))
	})

	t.Run("member passes the assignee check", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(9)).Return(domain.TeamMember{Role: domain.RoleMember}, nil)
		svc := New(tasks, members)

		require.NoError(t, svc.EnsureAssigneeMember(context.Background(), 3, 9))
	})

	t.Run("non-member assignee maps to client-visible validation error", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(9)).Return(domain.TeamMember{}, domain.ErrNotFound)
		svc := New(tasks, members)

		err := svc.EnsureAssigneeMember(context.Background(), 3, 9)

		require.ErrorIs(t, err, domain.ErrValidation)
		var safeErr *domain.SafeError
		require.ErrorAs(t, err, &safeErr)
		assert.Equal(t, "assignee is not a member of this team", safeErr.Msg)
	})

	t.Run("assignee membership repo failure is not a validation error", func(t *testing.T) {
		tasks := NewMockTaskGetter(t)
		members := NewMockMemberGetter(t)
		members.EXPECT().GetMember(mock.Anything, int64(3), int64(9)).Return(domain.TeamMember{}, dbErr)
		svc := New(tasks, members)

		err := svc.EnsureAssigneeMember(context.Background(), 3, 9)

		require.ErrorIs(t, err, dbErr)
		require.NotErrorIs(t, err, domain.ErrValidation)
	})
}
