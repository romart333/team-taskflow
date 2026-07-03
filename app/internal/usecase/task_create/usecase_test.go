package task_create

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type taskRepoMock struct {
	createID  int64
	createErr error
	task      domain.Task
	getErr    error
	gotTask   domain.Task
}

func (m *taskRepoMock) Create(_ context.Context, task domain.Task) (int64, error) {
	m.gotTask = task
	return m.createID, m.createErr
}

func (m *taskRepoMock) GetByID(context.Context, int64) (domain.Task, error) {
	return m.task, m.getErr
}

type teamRepoMock struct {
	// membership err per user id
	errs map[int64]error
}

func (m *teamRepoMock) GetMember(_ context.Context, _ int64, userID int64) (domain.TeamMember, error) {
	if err, ok := m.errs[userID]; ok {
		return domain.TeamMember{}, err
	}
	return domain.TeamMember{UserID: userID, Role: domain.RoleMember}, nil
}

type invalidatorMock struct {
	err   error
	calls int
}

func (m *invalidatorMock) InvalidateTeam(context.Context, int64) error {
	m.calls++
	return m.err
}

func TestUsecase_Handle(t *testing.T) {
	assignee := int64(9)

	tests := []struct {
		name    string
		input   Input
		tasks   *taskRepoMock
		teams   *teamRepoMock
		cache   *invalidatorMock
		wantErr error
	}{
		{
			name:  "success without assignee",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			tasks: &taskRepoMock{createID: 77, task: domain.Task{ID: 77, Title: "Fix bug"}},
			teams: &teamRepoMock{},
			cache: &invalidatorMock{},
		},
		{
			name:  "success with assignee member",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug", AssigneeID: &assignee},
			tasks: &taskRepoMock{createID: 77, task: domain.Task{ID: 77}},
			teams: &teamRepoMock{},
			cache: &invalidatorMock{},
		},
		{
			name:    "empty title",
			input:   Input{ActorID: 5, TeamID: 1, Title: ""},
			tasks:   &taskRepoMock{},
			teams:   &teamRepoMock{},
			cache:   &invalidatorMock{},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "author not a member",
			input:   Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			tasks:   &taskRepoMock{},
			teams:   &teamRepoMock{errs: map[int64]error{5: domain.ErrNotFound}},
			cache:   &invalidatorMock{},
			wantErr: domain.ErrPermissionDenied,
		},
		{
			name:    "assignee not a member",
			input:   Input{ActorID: 5, TeamID: 1, Title: "Fix bug", AssigneeID: &assignee},
			tasks:   &taskRepoMock{},
			teams:   &teamRepoMock{errs: map[int64]error{9: domain.ErrNotFound}},
			cache:   &invalidatorMock{},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "cache invalidation failure does not fail creation",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			tasks: &taskRepoMock{createID: 77, task: domain.Task{ID: 77}},
			teams: &teamRepoMock{},
			cache: &invalidatorMock{err: errors.New("redis down")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.tasks, tt.teams, tt.cache)

			out, err := uc.Handle(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Zero(t, tt.tasks.gotTask, "task must not be created on failure")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.tasks.task, out.Task)
			assert.Equal(t, domain.TaskStatusTodo, tt.tasks.gotTask.Status)
			assert.Equal(t, int64(5), tt.tasks.gotTask.CreatedBy)
			assert.Equal(t, 1, tt.cache.calls)
		})
	}
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	dbErr := errors.New("db down")
	assignee := int64(9)

	t.Run("membership check failure", func(t *testing.T) {
		teams := &teamRepoMock{errs: map[int64]error{5: dbErr}}
		uc := New(&taskRepoMock{}, teams, &invalidatorMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("assignee check failure", func(t *testing.T) {
		teams := &teamRepoMock{errs: map[int64]error{9: dbErr}}
		uc := New(&taskRepoMock{}, teams, &invalidatorMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t", AssigneeID: &assignee})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("create failure", func(t *testing.T) {
		uc := New(&taskRepoMock{createErr: dbErr}, &teamRepoMock{}, &invalidatorMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
	})

	t.Run("reload failure", func(t *testing.T) {
		uc := New(&taskRepoMock{createID: 1, getErr: dbErr}, &teamRepoMock{}, &invalidatorMock{})
		_, err := uc.Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
	})
}
