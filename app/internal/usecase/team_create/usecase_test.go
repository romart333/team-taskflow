package team_create

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type teamRepoMock struct {
	createID     int64
	createErr    error
	addMemberErr error
	gotTeam      domain.Team
	gotMember    domain.TeamMember
}

func (m *teamRepoMock) CreateTeam(_ context.Context, team domain.Team) (int64, error) {
	m.gotTeam = team
	return m.createID, m.createErr
}

func (m *teamRepoMock) AddMember(_ context.Context, member domain.TeamMember) error {
	m.gotMember = member
	return m.addMemberErr
}

type txMock struct{ called bool }

func (m *txMock) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	m.called = true
	return fn(ctx)
}

func TestUsecase_Handle(t *testing.T) {
	tests := []struct {
		name    string
		input   Input
		repo    *teamRepoMock
		wantErr error
	}{
		{
			name:  "success creates team with owner membership",
			input: Input{ActorID: 5, Name: "Platform"},
			repo:  &teamRepoMock{createID: 10},
		},
		{
			name:    "empty name",
			input:   Input{ActorID: 5, Name: ""},
			repo:    &teamRepoMock{},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "membership failure aborts",
			input:   Input{ActorID: 5, Name: "Platform"},
			repo:    &teamRepoMock{createID: 10, addMemberErr: errors.New("boom")},
			wantErr: nil, // generic error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transaction := &txMock{}
			uc := New(tt.repo, transaction)

			out, err := uc.Handle(context.Background(), tt.input)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.False(t, transaction.called, "validation must happen before the transaction")
			case tt.repo.addMemberErr != nil:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, Output{TeamID: 10, Name: "Platform"}, out)
				assert.True(t, transaction.called)
				assert.Equal(t, domain.Team{Name: "Platform", CreatedBy: 5}, tt.repo.gotTeam)
				assert.Equal(t, domain.TeamMember{TeamID: 10, UserID: 5, Role: domain.RoleOwner}, tt.repo.gotMember)
			}
		})
	}
}
