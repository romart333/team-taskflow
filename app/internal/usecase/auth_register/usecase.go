package auth_register

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return Output{}, fmt.Errorf("normalizing email: %w", err)
	}

	if err := domain.ValidateRegistration(in.Password, in.Name); err != nil {
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
		if errors.Is(err, domain.ErrAlreadyExists) {
			return Output{}, fmt.Errorf("creating user: %w",
				domain.NewAlreadyExistsError("user with this email already exists"))
		}
		return Output{}, fmt.Errorf("creating user: %w", err)
	}

	slog.InfoContext(ctx, "user registered", "user_id", userID)
	return Output{UserID: userID, Email: email, Name: in.Name}, nil
}
