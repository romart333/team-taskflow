package userrepo

import (
	"time"

	"team-taskflow/internal/domain"
)

type userEntity struct {
	ID           int64     `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (e userEntity) toDomain() domain.User {
	return domain.User{
		ID:           e.ID,
		Email:        e.Email,
		Name:         e.Name,
		PasswordHash: e.PasswordHash,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
