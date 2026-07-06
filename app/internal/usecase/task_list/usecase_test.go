package task_list

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
	pagination := Pagination{DefaultPageSize: 20, MaxPageSize: 100}
	dbPage := domain.TaskPage{Tasks: []domain.Task{{ID: 1, Title: "from db"}}, Total: 1}
	cachedPage := domain.TaskPage{Tasks: []domain.Task{{ID: 2, Title: "from cache"}}, Total: 1}
	normalizedFilter := domain.TaskFilter{TeamID: 1, Page: 1, PageSize: 20}

	t.Run("cache hit skips repository", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		cache.EXPECT().Get(mock.Anything, normalizedFilter).Return(cachedPage, true, int64(0), nil)

		out, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, cachedPage, out.Page)
	})

	t.Run("cache miss loads db and populates cache under the version seen at read time", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		cache.EXPECT().Get(mock.Anything, normalizedFilter).Return(domain.TaskPage{}, false, int64(7), nil)
		repo.EXPECT().List(mock.Anything, normalizedFilter).Return(dbPage, nil)
		cache.EXPECT().Set(mock.Anything, normalizedFilter, int64(7), dbPage).Return(nil)

		out, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, dbPage, out.Page)
	})

	t.Run("pagination is normalized", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		cache.EXPECT().Get(mock.Anything, mock.Anything).Return(domain.TaskPage{}, false, int64(0), nil)

		var gotFilter domain.TaskFilter
		repo.EXPECT().List(mock.Anything, mock.Anything).
			Run(func(_ context.Context, filter domain.TaskFilter) { gotFilter = filter }).
			Return(dbPage, nil)
		cache.EXPECT().Set(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		out, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1, Page: 0, PageSize: 1000}})

		require.NoError(t, err)
		assert.Equal(t, 1, gotFilter.Page)
		assert.Equal(t, 100, gotFilter.PageSize)
		assert.Equal(t, 1, out.PageNum)
		assert.Equal(t, 100, out.PageSize)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).
			Return(domain.NewPermissionDeniedError("not a member"))

		_, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("cache failures degrade to db and skip the cache write", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		cache.EXPECT().Get(mock.Anything, mock.Anything).
			Return(domain.TaskPage{}, false, int64(0), errors.New("redis down"))
		repo.EXPECT().List(mock.Anything, mock.Anything).Return(dbPage, nil)

		out, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, dbPage, out.Page)
	})

	t.Run("repository failure", func(t *testing.T) {
		repo := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		cache := NewMockTaskListCache(t)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		cache.EXPECT().Get(mock.Anything, mock.Anything).Return(domain.TaskPage{}, false, int64(0), nil)
		repo.EXPECT().List(mock.Anything, mock.Anything).Return(domain.TaskPage{}, errors.New("db down"))

		_, err := New(repo, access, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.Error(t, err)
	})
}
