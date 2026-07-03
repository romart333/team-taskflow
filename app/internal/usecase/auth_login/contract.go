package auth_login

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	Email    string
	Password string
}

type Output struct {
	AccessToken string
}

// UserRepository is the read port for user accounts.
type UserRepository interface {
	// GetByEmail returns domain.ErrNotFound when no such user exists.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

// PasswordVerifier compares a stored hash with a plaintext candidate.
type PasswordVerifier interface {
	Verify(hash, password string) error
}

// TokenIssuer issues signed access tokens for authenticated users.
type TokenIssuer interface {
	Issue(userID int64) (string, error)
}
