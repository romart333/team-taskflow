package teamrepo

import (
	"time"

	"team-taskflow/internal/domain"
)

type teamEntity struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedBy int64     `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

func (e teamEntity) toDomain() domain.Team {
	return domain.Team{
		ID:        e.ID,
		Name:      e.Name,
		CreatedBy: e.CreatedBy,
		CreatedAt: e.CreatedAt,
	}
}

type memberEntity struct {
	TeamID   int64     `db:"team_id"`
	UserID   int64     `db:"user_id"`
	Role     string    `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}

func (e memberEntity) toDomain() domain.TeamMember {
	return domain.TeamMember{
		TeamID:   e.TeamID,
		UserID:   e.UserID,
		Role:     domain.Role(e.Role),
		JoinedAt: e.JoinedAt,
	}
}
