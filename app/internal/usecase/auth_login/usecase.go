package auth_login

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	users    UserRepository
	verifier PasswordVerifier
	tokens   TokenIssuer
}

func New(users UserRepository, verifier PasswordVerifier, tokens TokenIssuer) *Usecase {
	return &Usecase{users: users, verifier: verifier, tokens: tokens}
}

func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := u.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Deliberately indistinguishable from a wrong password.
			return Output{}, fmt.Errorf("looking up user: %w", domain.ErrUnauthorized)
		}
		return Output{}, fmt.Errorf("looking up user by email: %w", err)
	}

	if err := u.verifier.Verify(user.PasswordHash, in.Password); err != nil {
		return Output{}, fmt.Errorf("verifying password: %w", domain.ErrUnauthorized)
	}

	token, err := u.tokens.Issue(user.ID)
	if err != nil {
		return Output{}, fmt.Errorf("issuing token: %w", err)
	}

	slog.InfoContext(ctx, "user logged in", "user_id", user.ID)
	return Output{AccessToken: token}, nil
}
