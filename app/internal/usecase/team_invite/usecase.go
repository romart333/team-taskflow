package team_invite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	teams    TeamRepository
	users    UserRepository
	notifier InviteNotifier
}

func New(teams TeamRepository, users UserRepository, notifier InviteNotifier) *Usecase {
	return &Usecase{teams: teams, users: users, notifier: notifier}
}

// Handle adds an existing user to the team. Only owners and admins may invite.
// The email notification is best effort: its failure never fails the invite.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	role, err := u.inviteRole(in.Role)
	if err != nil {
		return Output{}, fmt.Errorf("validating invite role: %w", err)
	}

	actor, err := u.teams.GetMember(ctx, in.TeamID, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("checking actor membership: %w",
				domain.NewPermissionDeniedError("you are not a member of this team"))
		}
		return Output{}, fmt.Errorf("getting actor membership: %w", err)
	}
	if !actor.Role.CanInvite() {
		return Output{}, fmt.Errorf("actor role %q: %w", actor.Role,
			domain.NewPermissionDeniedError("only team owners and admins can invite"))
	}

	team, err := u.teams.GetTeam(ctx, in.TeamID)
	if err != nil {
		return Output{}, fmt.Errorf("getting team: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	invitee, err := u.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("looking up invitee: %w",
				domain.NewNotFoundError("user with this email is not registered"))
		}
		return Output{}, fmt.Errorf("getting invitee by email: %w", err)
	}

	if err := u.teams.AddMember(ctx, domain.TeamMember{
		TeamID: in.TeamID,
		UserID: invitee.ID,
		Role:   role,
	}); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return Output{}, fmt.Errorf("adding member: %w",
				domain.NewConflictError("user is already a member of this team"))
		}
		return Output{}, fmt.Errorf("adding member: %w", err)
	}

	if err := u.notifier.SendInvite(ctx, email, team.Name); err != nil {
		slog.WarnContext(ctx, "invite notification failed",
			"team_id", in.TeamID, "user_id", invitee.ID, "error", err)
	}

	slog.InfoContext(ctx, "user invited to team",
		"team_id", in.TeamID, "user_id", invitee.ID, "role", role, "invited_by", in.ActorID)
	return Output{TeamID: in.TeamID, UserID: invitee.ID, Role: role}, nil
}

func (u *Usecase) inviteRole(raw string) (domain.Role, error) {
	if raw == "" {
		return domain.RoleMember, nil
	}
	role, err := domain.ParseRole(raw)
	if err != nil {
		return "", err
	}
	if role == domain.RoleOwner {
		return "", domain.NewValidationError("cannot invite a user as owner")
	}
	return role, nil
}
