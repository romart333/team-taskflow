package task_create

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
	assignee := int64(9)
	created := domain.Task{ID: 77, Title: "Fix bug"}

	tests := []struct {
		name    string
		input   Input
		setup   func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator)
		wantErr error
	}{
		{
			name:  "success without assignee",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			setup: func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator) {
				access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
				tasks.EXPECT().Create(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
					return task.Status == domain.TaskStatusTodo && task.CreatedBy == 5
				})).Return(77, nil)
				tasks.EXPECT().GetByID(mock.Anything, int64(77)).Return(created, nil)
				cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil)
			},
		},
		{
			name:  "success with assignee member",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug", AssigneeID: &assignee},
			setup: func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator) {
				access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
				access.EXPECT().EnsureAssigneeMember(mock.Anything, int64(1), assignee).Return(nil)
				tasks.EXPECT().Create(mock.Anything, mock.Anything).Return(77, nil)
				tasks.EXPECT().GetByID(mock.Anything, int64(77)).Return(created, nil)
				cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil)
			},
		},
		{
			name:    "empty title",
			input:   Input{ActorID: 5, TeamID: 1, Title: ""},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "author not a member",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			setup: func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator) {
				access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).
					Return(domain.NewPermissionDeniedError("not a member"))
			},
			wantErr: domain.ErrPermissionDenied,
		},
		{
			name:  "assignee not a member",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug", AssigneeID: &assignee},
			setup: func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator) {
				access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
				access.EXPECT().EnsureAssigneeMember(mock.Anything, int64(1), assignee).
					Return(domain.NewValidationError("assignee is not a member of this team"))
			},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "cache invalidation failure does not fail creation",
			input: Input{ActorID: 5, TeamID: 1, Title: "Fix bug"},
			setup: func(tasks *MockTaskRepository, access *MockTeamAccess, cache *MockTaskCacheInvalidator) {
				access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
				tasks.EXPECT().Create(mock.Anything, mock.Anything).Return(77, nil)
				tasks.EXPECT().GetByID(mock.Anything, int64(77)).Return(created, nil)
				cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(errors.New("redis down"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := NewMockTaskRepository(t)
			access := NewMockTeamAccess(t)
			cache := NewMockTaskCacheInvalidator(t)
			if tt.setup != nil {
				tt.setup(tasks, access, cache)
			}
			uc := New(tasks, access, cache)

			out, err := uc.Handle(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, created, out.Task)
		})
	}
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	dbErr := errors.New("db down")
	assignee := int64(9)

	t.Run("membership check failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskCacheInvalidator(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(dbErr)

		_, err := New(tasks, access, cache).Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("assignee check failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskCacheInvalidator(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		access.EXPECT().EnsureAssigneeMember(mock.Anything, int64(1), assignee).Return(dbErr)

		_, err := New(tasks, access, cache).Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t", AssigneeID: &assignee})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("create failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskCacheInvalidator(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		tasks.EXPECT().Create(mock.Anything, mock.Anything).Return(0, dbErr)

		_, err := New(tasks, access, cache).Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
	})

	t.Run("reload failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskCacheInvalidator(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		tasks.EXPECT().Create(mock.Anything, mock.Anything).Return(1, nil)
		tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(domain.Task{}, dbErr)

		_, err := New(tasks, access, cache).Handle(context.Background(), Input{ActorID: 5, TeamID: 1, Title: "t"})
		require.Error(t, err)
	})
}
