package auth_register

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	users  UserRepository
	hasher PasswordHasher
}

func New(users UserRepository, hasher PasswordHasher) *Usecase {
	return &Usecase{users: users, hasher: hasher}
}

func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if err := domain.ValidateRegistration(email, in.Password, in.Name); err != nil {
		return Output{}, fmt.Errorf("validating registration: %w", err)
	}

	hash, err := u.hasher.Hash(in.Password)
	if err != nil {
		return Output{}, fmt.Errorf("hashing password: %w", err)
	}

	userID, err := u.users.Create(ctx, domain.User{
		Email:        email,
		Name:         in.Name,
		PasswordHash: hash,
	})
	if err != nil {
		return Output{}, fmt.Errorf("creating user: %w", err)
	}

	slog.InfoContext(ctx, "user registered", "user_id", userID)
	return Output{UserID: userID, Email: email, Name: in.Name}, nil
}
