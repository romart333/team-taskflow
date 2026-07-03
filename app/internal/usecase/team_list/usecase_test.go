package team_list

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type teamRepoMock struct {
	teams     []domain.TeamWithRole
	err       error
	gotUserID int64
}

func (m *teamRepoMock) ListByUser(_ context.Context, userID int64) ([]domain.TeamWithRole, error) {
	m.gotUserID = userID
	return m.teams, m.err
}

func TestUsecase_Handle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := []domain.TeamWithRole{
			{Team: domain.Team{ID: 1, Name: "Platform"}, Role: domain.RoleOwner},
		}
		repo := &teamRepoMock{teams: expected}

		out, err := New(repo).Handle(context.Background(), Input{ActorID: 5})

		require.NoError(t, err)
		assert.Equal(t, expected, out.Teams)
		assert.Equal(t, int64(5), repo.gotUserID)
	})

	t.Run("repository failure", func(t *testing.T) {
		repo := &teamRepoMock{err: errors.New("db down")}

		_, err := New(repo).Handle(context.Background(), Input{ActorID: 5})

		require.Error(t, err)
	})
}
