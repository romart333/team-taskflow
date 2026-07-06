package analytics_get

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

func TestUsecase(t *testing.T) {
	const actorID int64 = 42

	t.Run("team stats uses domain window and scopes to the actor", func(t *testing.T) {
		repo := NewMockAnalyticsRepository(t)
		repo.EXPECT().TeamStats(mock.Anything, actorID, domain.TeamStatsDoneWindowDays).
			Return([]domain.TeamStats{{TeamID: 1, MemberCount: 3}}, nil)

		stats, err := New(repo).TeamStats(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, stats, 1)
	})

	t.Run("top creators uses domain window and limit and scopes to the actor", func(t *testing.T) {
		repo := NewMockAnalyticsRepository(t)
		repo.EXPECT().TopCreators(mock.Anything, actorID, domain.TopCreatorsWindowDays, domain.TopCreatorsLimit).
			Return([]domain.TeamTopCreator{{TeamID: 1, Rank: 1}}, nil)

		creators, err := New(repo).TopCreators(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, creators, 1)
	})

	t.Run("orphaned assignees scopes to the actor", func(t *testing.T) {
		repo := NewMockAnalyticsRepository(t)
		repo.EXPECT().OrphanedAssignees(mock.Anything, actorID).
			Return([]domain.OrphanedAssigneeTask{{TaskID: 9}}, nil)

		tasks, err := New(repo).OrphanedAssignees(context.Background(), actorID)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
	})

	t.Run("errors are wrapped", func(t *testing.T) {
		repo := NewMockAnalyticsRepository(t)
		dbErr := errors.New("db down")
		repo.EXPECT().TeamStats(mock.Anything, actorID, domain.TeamStatsDoneWindowDays).Return(nil, dbErr)
		repo.EXPECT().TopCreators(mock.Anything, actorID, domain.TopCreatorsWindowDays, domain.TopCreatorsLimit).Return(nil, dbErr)
		repo.EXPECT().OrphanedAssignees(mock.Anything, actorID).Return(nil, dbErr)

		_, err := New(repo).TeamStats(context.Background(), actorID)
		require.Error(t, err)
		_, err = New(repo).TopCreators(context.Background(), actorID)
		require.Error(t, err)
		_, err = New(repo).OrphanedAssignees(context.Background(), actorID)
		require.Error(t, err)
	})
}
