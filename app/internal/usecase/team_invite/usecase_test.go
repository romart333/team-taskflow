package team_invite

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
	ownerMember := domain.TeamMember{TeamID: 1, UserID: 5, Role: domain.RoleOwner}

	tests := []struct {
		name     string
		input    Input
		setup    func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember
		wantErr  error
		wantRole domain.Role
	}{
		{
			name:  "owner invites with default member role",
			input: Input{ActorID: 5, TeamID: 1, Email: "Bob@Example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(ownerMember, nil)
				teams.EXPECT().GetTeam(mock.Anything, int64(1)).Return(domain.Team{ID: 1, Name: "Platform"}, nil)
				users.EXPECT().GetByEmail(mock.Anything, "bob@example.com").Return(domain.User{ID: 9}, nil)
				var gotMember domain.TeamMember
				teams.EXPECT().AddMember(mock.Anything, mock.MatchedBy(func(member domain.TeamMember) bool {
					gotMember = member
					return true
				})).Return(nil).Once()
				notifier.EXPECT().SendInvite(mock.Anything, "bob@example.com", "Platform").Return(nil).Once()
				return &gotMember
			},
			wantRole: domain.RoleMember,
		},
		{
			name:  "admin invites as admin",
			input: Input{ActorID: 5, TeamID: 1, Email: "bob@example.com", Role: "admin"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(domain.TeamMember{Role: domain.RoleAdmin}, nil)
				teams.EXPECT().GetTeam(mock.Anything, int64(1)).Return(domain.Team{Name: "Platform"}, nil)
				users.EXPECT().GetByEmail(mock.Anything, "bob@example.com").Return(domain.User{ID: 9}, nil)
				var gotMember domain.TeamMember
				teams.EXPECT().AddMember(mock.Anything, mock.MatchedBy(func(member domain.TeamMember) bool {
					gotMember = member
					return true
				})).Return(nil).Once()
				notifier.EXPECT().SendInvite(mock.Anything, "bob@example.com", "Platform").Return(nil).Once()
				return &gotMember
			},
			wantRole: domain.RoleAdmin,
		},
		{
			name:  "member cannot invite",
			input: Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(domain.TeamMember{Role: domain.RoleMember}, nil)
				return nil
			},
			wantErr: domain.ErrPermissionDenied,
		},
		{
			name:  "non-member cannot invite",
			input: Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(domain.TeamMember{}, domain.ErrNotFound)
				return nil
			},
			wantErr: domain.ErrPermissionDenied,
		},
		{
			name:    "cannot invite as owner",
			input:   Input{ActorID: 5, TeamID: 1, Email: "bob@example.com", Role: "owner"},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "invitee not registered",
			input: Input{ActorID: 5, TeamID: 1, Email: "ghost@example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(ownerMember, nil)
				teams.EXPECT().GetTeam(mock.Anything, int64(1)).Return(domain.Team{Name: "Platform"}, nil)
				users.EXPECT().GetByEmail(mock.Anything, "ghost@example.com").Return(domain.User{}, domain.ErrNotFound)
				return nil
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "already a member",
			input: Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(ownerMember, nil)
				teams.EXPECT().GetTeam(mock.Anything, int64(1)).Return(domain.Team{Name: "Platform"}, nil)
				users.EXPECT().GetByEmail(mock.Anything, "bob@example.com").Return(domain.User{ID: 9}, nil)
				teams.EXPECT().AddMember(mock.Anything, mock.Anything).Return(domain.ErrAlreadyExists)
				return nil
			},
			wantErr: domain.ErrConflict,
		},
		{
			name:  "notification failure does not fail the invite",
			input: Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			setup: func(teams *MockTeamRepository, users *MockUserRepository, notifier *MockInviteNotifier) *domain.TeamMember {
				teams.EXPECT().GetMember(mock.Anything, int64(1), int64(5)).Return(ownerMember, nil)
				teams.EXPECT().GetTeam(mock.Anything, int64(1)).Return(domain.Team{Name: "Platform"}, nil)
				users.EXPECT().GetByEmail(mock.Anything, "bob@example.com").Return(domain.User{ID: 9}, nil)
				var gotMember domain.TeamMember
				teams.EXPECT().AddMember(mock.Anything, mock.MatchedBy(func(member domain.TeamMember) bool {
					gotMember = member
					return true
				})).Return(nil).Once()
				notifier.EXPECT().SendInvite(mock.Anything, "bob@example.com", "Platform").
					Return(errors.New("email service down")).Once()
				return &gotMember
			},
			wantRole: domain.RoleMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teams := NewMockTeamRepository(t)
			users := NewMockUserRepository(t)
			notifier := NewMockInviteNotifier(t)
			var gotMember *domain.TeamMember
			if tt.setup != nil {
				gotMember = tt.setup(teams, users, notifier)
			}
			uc := New(teams, users, notifier)

			out, err := uc.Handle(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRole, out.Role)
			require.NotNil(t, gotMember)
			assert.Equal(t, tt.wantRole, gotMember.Role)
			assert.Equal(t, int64(9), out.UserID)
		})
	}
}
