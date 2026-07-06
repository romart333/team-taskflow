package auth_register

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	Email    string
	Password string
	Name     string
}

type Output struct {
	UserID int64
	Email  string
	Name   string
}

// UserRepository is the persistence port for user accounts.
type UserRepository interface {
	// Create inserts the user and returns its ID. It returns
	// domain.ErrAlreadyExists when the email is already taken.
	Create(ctx context.Context, user domain.User) (int64, error)
}

// PasswordHasher hashes plaintext passwords for storage.
type PasswordHasher interface {
	Hash(password string) (string, error)
}
