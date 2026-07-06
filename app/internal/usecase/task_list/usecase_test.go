package task_list

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type taskRepoMock struct {
	page      domain.TaskPage
	err       error
	calls     int
	gotFilter domain.TaskFilter
}

func (m *taskRepoMock) List(_ context.Context, filter domain.TaskFilter) (domain.TaskPage, error) {
	m.calls++
	m.gotFilter = filter
	return m.page, m.err
}

type accessMock struct{ err error }

func (m *accessMock) EnsureTeamMember(context.Context, int64, int64) error {
	return m.err
}

type cacheMock struct {
	page       domain.TaskPage
	hit        bool
	version    int64
	getErr     error
	setErr     error
	setCalls   int
	gotSet     domain.TaskPage
	gotVersion int64
}

func (m *cacheMock) Get(context.Context, domain.TaskFilter) (domain.TaskPage, bool, int64, error) {
	return m.page, m.hit, m.version, m.getErr
}

func (m *cacheMock) Set(_ context.Context, _ domain.TaskFilter, version int64, page domain.TaskPage) error {
	m.setCalls++
	m.gotVersion = version
	m.gotSet = page
	return m.setErr
}

func TestUsecase_Handle(t *testing.T) {
	pagination := Pagination{DefaultPageSize: 20, MaxPageSize: 100}
	dbPage := domain.TaskPage{Tasks: []domain.Task{{ID: 1, Title: "from db"}}, Total: 1}
	cachedPage := domain.TaskPage{Tasks: []domain.Task{{ID: 2, Title: "from cache"}}, Total: 1}

	t.Run("cache hit skips repository", func(t *testing.T) {
		repo := &taskRepoMock{page: dbPage}
		cache := &cacheMock{page: cachedPage, hit: true}

		out, err := New(repo, &accessMock{}, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, cachedPage, out.Page)
		assert.Zero(t, repo.calls)
	})

	t.Run("cache miss loads db and populates cache under the version seen at read time", func(t *testing.T) {
		repo := &taskRepoMock{page: dbPage}
		cache := &cacheMock{version: 7}

		out, err := New(repo, &accessMock{}, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, dbPage, out.Page)
		assert.Equal(t, 1, repo.calls)
		assert.Equal(t, 1, cache.setCalls)
		assert.Equal(t, dbPage, cache.gotSet)
		assert.EqualValues(t, 7, cache.gotVersion, "Set must reuse the version observed by Get")
	})

	t.Run("pagination is normalized", func(t *testing.T) {
		repo := &taskRepoMock{page: dbPage}
		cache := &cacheMock{}

		out, err := New(repo, &accessMock{}, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1, Page: 0, PageSize: 1000}})

		require.NoError(t, err)
		assert.Equal(t, 1, repo.gotFilter.Page)
		assert.Equal(t, 100, repo.gotFilter.PageSize)
		assert.Equal(t, 1, out.PageNum)
		assert.Equal(t, 100, out.PageSize)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		_, err := New(&taskRepoMock{}, &accessMock{err: domain.NewPermissionDeniedError("not a member")}, &cacheMock{}, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("cache failures degrade to db and skip the cache write", func(t *testing.T) {
		repo := &taskRepoMock{page: dbPage}
		cache := &cacheMock{getErr: errors.New("redis down"), setErr: errors.New("redis down")}

		out, err := New(repo, &accessMock{}, cache, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.NoError(t, err)
		assert.Equal(t, dbPage, out.Page)
		assert.Zero(t, cache.setCalls, "an unversioned page must not be written to the cache")
	})

	t.Run("repository failure", func(t *testing.T) {
		repo := &taskRepoMock{err: errors.New("db down")}

		_, err := New(repo, &accessMock{}, &cacheMock{}, pagination).
			Handle(context.Background(), Input{ActorID: 5, Filter: domain.TaskFilter{TeamID: 1}})

		require.Error(t, err)
	})
}
