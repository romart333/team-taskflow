package domain

import (
	"fmt"
	"time"
	"unicode/utf8"
)

// Role is a member's role inside a team.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// ParseRole validates a raw role string coming from the outside.
func ParseRole(raw string) (Role, error) {
	switch Role(raw) {
	case RoleOwner, RoleAdmin, RoleMember:
		return Role(raw), nil
	default:
		return "", NewValidationError("role must be one of: owner, admin, member")
	}
}

// CanInvite reports whether a member with this role may invite users.
func (r Role) CanInvite() bool {
	return r == RoleOwner || r == RoleAdmin
}

type Team struct {
	ID        int64
	Name      string
	CreatedBy int64
	CreatedAt time.Time
}

// MaxTeamNameLength mirrors the teams.name VARCHAR(255) column.
const MaxTeamNameLength = 255

// ValidateTeamName checks the invariants for a new team.
func ValidateTeamName(name string) error {
	if name == "" {
		return NewValidationError("team name is required")
	}
	if utf8.RuneCountInString(name) > MaxTeamNameLength {
		return NewValidationError(fmt.Sprintf("team name must be at most %d characters", MaxTeamNameLength))
	}
	return nil
}

type TeamMember struct {
	TeamID   int64
	UserID   int64
	Role     Role
	JoinedAt time.Time
}

// TeamWithRole is a team together with the requesting user's role in it.
type TeamWithRole struct {
	Team Team
	Role Role
}
