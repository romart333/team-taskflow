package team_invite

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type teamRepoMock struct {
	member       domain.TeamMember
	memberErr    error
	team         domain.Team
	teamErr      error
	addMemberErr error
	gotMember    domain.TeamMember
}

func (m *teamRepoMock) GetTeam(context.Context, int64) (domain.Team, error) {
	return m.team, m.teamErr
}

func (m *teamRepoMock) GetMember(context.Context, int64, int64) (domain.TeamMember, error) {
	return m.member, m.memberErr
}

func (m *teamRepoMock) AddMember(_ context.Context, member domain.TeamMember) error {
	m.gotMember = member
	return m.addMemberErr
}

type userRepoMock struct {
	user domain.User
	err  error
}

func (m *userRepoMock) GetByEmail(context.Context, string) (domain.User, error) {
	return m.user, m.err
}

type notifierMock struct {
	err      error
	gotEmail string
	gotTeam  string
	calls    int
}

func (m *notifierMock) SendInvite(_ context.Context, email, teamName string) error {
	m.calls++
	m.gotEmail = email
	m.gotTeam = teamName
	return m.err
}

func TestUsecase_Handle(t *testing.T) {
	ownerMember := domain.TeamMember{TeamID: 1, UserID: 5, Role: domain.RoleOwner}

	tests := []struct {
		name     string
		input    Input
		teams    *teamRepoMock
		users    *userRepoMock
		notifier *notifierMock
		wantErr  error
		wantRole domain.Role
	}{
		{
			name:     "owner invites with default member role",
			input:    Input{ActorID: 5, TeamID: 1, Email: "Bob@Example.com"},
			teams:    &teamRepoMock{member: ownerMember, team: domain.Team{ID: 1, Name: "Platform"}},
			users:    &userRepoMock{user: domain.User{ID: 9}},
			notifier: &notifierMock{},
			wantRole: domain.RoleMember,
		},
		{
			name:     "admin invites as admin",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com", Role: "admin"},
			teams:    &teamRepoMock{member: domain.TeamMember{Role: domain.RoleAdmin}, team: domain.Team{Name: "Platform"}},
			users:    &userRepoMock{user: domain.User{ID: 9}},
			notifier: &notifierMock{},
			wantRole: domain.RoleAdmin,
		},
		{
			name:     "member cannot invite",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			teams:    &teamRepoMock{member: domain.TeamMember{Role: domain.RoleMember}},
			users:    &userRepoMock{},
			notifier: &notifierMock{},
			wantErr:  domain.ErrPermissionDenied,
		},
		{
			name:     "non-member cannot invite",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			teams:    &teamRepoMock{memberErr: domain.ErrNotFound},
			users:    &userRepoMock{},
			notifier: &notifierMock{},
			wantErr:  domain.ErrPermissionDenied,
		},
		{
			name:     "cannot invite as owner",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com", Role: "owner"},
			teams:    &teamRepoMock{member: ownerMember},
			users:    &userRepoMock{},
			notifier: &notifierMock{},
			wantErr:  domain.ErrValidation,
		},
		{
			name:     "invitee not registered",
			input:    Input{ActorID: 5, TeamID: 1, Email: "ghost@example.com"},
			teams:    &teamRepoMock{member: ownerMember, team: domain.Team{Name: "Platform"}},
			users:    &userRepoMock{err: domain.ErrNotFound},
			notifier: &notifierMock{},
			wantErr:  domain.ErrNotFound,
		},
		{
			name:     "already a member",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			teams:    &teamRepoMock{member: ownerMember, team: domain.Team{Name: "Platform"}, addMemberErr: domain.ErrAlreadyExists},
			users:    &userRepoMock{user: domain.User{ID: 9}},
			notifier: &notifierMock{},
			wantErr:  domain.ErrConflict,
		},
		{
			name:     "notification failure does not fail the invite",
			input:    Input{ActorID: 5, TeamID: 1, Email: "bob@example.com"},
			teams:    &teamRepoMock{member: ownerMember, team: domain.Team{Name: "Platform"}},
			users:    &userRepoMock{user: domain.User{ID: 9}},
			notifier: &notifierMock{err: errors.New("email service down")},
			wantRole: domain.RoleMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.teams, tt.users, tt.notifier)

			out, err := uc.Handle(context.Background(), tt.input)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRole, out.Role)
			assert.Equal(t, tt.wantRole, tt.teams.gotMember.Role)
			assert.Equal(t, int64(9), out.UserID)
			assert.Equal(t, 1, tt.notifier.calls)
			assert.Equal(t, "bob@example.com", tt.notifier.gotEmail, "email must be normalized")
			assert.Equal(t, "Platform", tt.notifier.gotTeam)
		})
	}
}
