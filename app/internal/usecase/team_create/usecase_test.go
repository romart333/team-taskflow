package team_create

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
	tests := []struct {
		name          string
		input         Input
		setup         func(repo *MockTeamRepository, tx *MockTxManager)
		wantErr       error
		wantAddMember bool
		want          Output
	}{
		{
			name:  "success creates team with owner membership",
			input: Input{ActorID: 5, Name: "Platform"},
			setup: func(repo *MockTeamRepository, tx *MockTxManager) {
				tx.EXPECT().Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})
				repo.EXPECT().CreateTeam(mock.Anything, domain.Team{Name: "Platform", CreatedBy: 5}).Return(10, nil)
				repo.EXPECT().AddMember(mock.Anything, domain.TeamMember{TeamID: 10, UserID: 5, Role: domain.RoleOwner}).Return(nil)
			},
			want: Output{TeamID: 10, Name: "Platform"},
		},
		{
			name:    "empty name",
			input:   Input{ActorID: 5, Name: ""},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "membership failure aborts",
			input: Input{ActorID: 5, Name: "Platform"},
			setup: func(repo *MockTeamRepository, tx *MockTxManager) {
				tx.EXPECT().Do(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})
				repo.EXPECT().CreateTeam(mock.Anything, domain.Team{Name: "Platform", CreatedBy: 5}).Return(10, nil)
				repo.EXPECT().AddMember(mock.Anything, domain.TeamMember{TeamID: 10, UserID: 5, Role: domain.RoleOwner}).
					Return(errors.New("boom"))
			},
			wantAddMember: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTeamRepository(t)
			transaction := NewMockTxManager(t)
			if tt.setup != nil {
				tt.setup(repo, transaction)
			}
			uc := New(repo, transaction)

			out, err := uc.Handle(context.Background(), tt.input)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantAddMember:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.want, out)
			}
		})
	}
}
