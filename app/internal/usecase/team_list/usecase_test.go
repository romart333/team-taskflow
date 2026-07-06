package team_list

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
	t.Run("success", func(t *testing.T) {
		expected := []domain.TeamWithRole{
			{Team: domain.Team{ID: 1, Name: "Platform"}, Role: domain.RoleOwner},
		}
		repo := NewMockTeamRepository(t)
		repo.EXPECT().ListByUser(mock.Anything, int64(5)).Return(expected, nil)

		out, err := New(repo).Handle(context.Background(), Input{ActorID: 5})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Teams)
	})

	t.Run("repository failure", func(t *testing.T) {
		repo := NewMockTeamRepository(t)
		repo.EXPECT().ListByUser(mock.Anything, int64(5)).Return(nil, errors.New("db down"))

		_, err := New(repo).Handle(context.Background(), Input{ActorID: 5})

		require.Error(t, err)
	})
}
