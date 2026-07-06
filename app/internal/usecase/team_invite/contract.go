package team_invite

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	TeamID  int64
	Email   string
	// Role is optional; empty means domain.RoleMember.
	Role string
}

type Output struct {
	TeamID int64
	UserID int64
	Role   domain.Role
}

// TeamRepository is the persistence port for teams and memberships.
type TeamRepository interface {
	// GetTeam returns domain.ErrNotFound when the team does not exist.
	GetTeam(ctx context.Context, teamID int64) (domain.Team, error)
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
	// AddMember returns domain.ErrAlreadyExists when the membership exists.
	AddMember(ctx context.Context, member domain.TeamMember) error
}

// UserRepository is the read port for user accounts.
type UserRepository interface {
	// GetByEmail returns domain.ErrNotFound when no such user exists.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

// InviteNotifier delivers invite notifications (best effort).
type InviteNotifier interface {
	SendInvite(ctx context.Context, email, teamName string) error
}
