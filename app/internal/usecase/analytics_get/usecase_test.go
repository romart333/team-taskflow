package analytics_get

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type analyticsRepoMock struct {
	stats    []domain.TeamStats
	creators []domain.TeamTopCreator
	orphaned []domain.OrphanedAssigneeTask
	err      error

	gotWindowDays int
	gotLimit      int
}

func (m *analyticsRepoMock) TeamStats(_ context.Context, doneWindowDays int) ([]domain.TeamStats, error) {
	m.gotWindowDays = doneWindowDays
	return m.stats, m.err
}

func (m *analyticsRepoMock) TopCreators(_ context.Context, windowDays, limit int) ([]domain.TeamTopCreator, error) {
	m.gotWindowDays = windowDays
	m.gotLimit = limit
	return m.creators, m.err
}

func (m *analyticsRepoMock) OrphanedAssignees(context.Context) ([]domain.OrphanedAssigneeTask, error) {
	return m.orphaned, m.err
}

func TestUsecase(t *testing.T) {
	t.Run("team stats uses domain window", func(t *testing.T) {
		repo := &analyticsRepoMock{stats: []domain.TeamStats{{TeamID: 1, MemberCount: 3}}}

		stats, err := New(repo).TeamStats(context.Background())

		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.Equal(t, domain.TeamStatsDoneWindowDays, repo.gotWindowDays)
	})

	t.Run("top creators uses domain window and limit", func(t *testing.T) {
		repo := &analyticsRepoMock{creators: []domain.TeamTopCreator{{TeamID: 1, Rank: 1}}}

		creators, err := New(repo).TopCreators(context.Background())

		require.NoError(t, err)
		assert.Len(t, creators, 1)
		assert.Equal(t, domain.TopCreatorsWindowDays, repo.gotWindowDays)
		assert.Equal(t, domain.TopCreatorsLimit, repo.gotLimit)
	})

	t.Run("orphaned assignees", func(t *testing.T) {
		repo := &analyticsRepoMock{orphaned: []domain.OrphanedAssigneeTask{{TaskID: 9}}}

		tasks, err := New(repo).OrphanedAssignees(context.Background())

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
	})

	t.Run("errors are wrapped", func(t *testing.T) {
		repo := &analyticsRepoMock{err: errors.New("db down")}

		_, err := New(repo).TeamStats(context.Background())
		require.Error(t, err)
		_, err = New(repo).TopCreators(context.Background())
		require.Error(t, err)
		_, err = New(repo).OrphanedAssignees(context.Background())
		require.Error(t, err)
	})
}
