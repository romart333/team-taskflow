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

	gotActorID    int64
	gotWindowDays int
	gotLimit      int
}

func (m *analyticsRepoMock) TeamStats(_ context.Context, actorID int64, doneWindowDays int) ([]domain.TeamStats, error) {
	m.gotActorID = actorID
	m.gotWindowDays = doneWindowDays
	return m.stats, m.err
}

func (m *analyticsRepoMock) TopCreators(_ context.Context, actorID int64, windowDays, limit int) ([]domain.TeamTopCreator, error) {
	m.gotActorID = actorID
	m.gotWindowDays = windowDays
	m.gotLimit = limit
	return m.creators, m.err
}

func (m *analyticsRepoMock) OrphanedAssignees(_ context.Context, actorID int64) ([]domain.OrphanedAssigneeTask, error) {
	m.gotActorID = actorID
	return m.orphaned, m.err
}

func TestUsecase(t *testing.T) {
	const actorID int64 = 42

	t.Run("team stats uses domain window and scopes to the actor", func(t *testing.T) {
		repo := &analyticsRepoMock{stats: []domain.TeamStats{{TeamID: 1, MemberCount: 3}}}

		stats, err := New(repo).TeamStats(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.Equal(t, domain.TeamStatsDoneWindowDays, repo.gotWindowDays)
		assert.Equal(t, actorID, repo.gotActorID)
	})

	t.Run("top creators uses domain window and limit and scopes to the actor", func(t *testing.T) {
		repo := &analyticsRepoMock{creators: []domain.TeamTopCreator{{TeamID: 1, Rank: 1}}}

		creators, err := New(repo).TopCreators(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, creators, 1)
		assert.Equal(t, domain.TopCreatorsWindowDays, repo.gotWindowDays)
		assert.Equal(t, domain.TopCreatorsLimit, repo.gotLimit)
		assert.Equal(t, actorID, repo.gotActorID)
	})

	t.Run("orphaned assignees scopes to the actor", func(t *testing.T) {
		repo := &analyticsRepoMock{orphaned: []domain.OrphanedAssigneeTask{{TaskID: 9}}}

		tasks, err := New(repo).OrphanedAssignees(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, actorID, repo.gotActorID)
	})

	t.Run("errors are wrapped", func(t *testing.T) {
		repo := &analyticsRepoMock{err: errors.New("db down")}

		_, err := New(repo).TeamStats(context.Background(), actorID)
		require.Error(t, err)
		_, err = New(repo).TopCreators(context.Background(), actorID)
		require.Error(t, err)
		_, err = New(repo).OrphanedAssignees(context.Background(), actorID)
		require.Error(t, err)
	})
}
