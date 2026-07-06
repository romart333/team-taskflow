package auth_register

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type userRepoMock struct {
	createID  int64
	createErr error
	gotUser   domain.User
}

func (m *userRepoMock) Create(_ context.Context, user domain.User) (int64, error) {
	m.gotUser = user
	return m.createID, m.createErr
}

type hasherMock struct {
	hash string
	err  error
}

func (m *hasherMock) Hash(string) (string, error) { return m.hash, m.err }

func TestUsecase_Handle(t *testing.T) {
	tests := []struct {
		name      string
		input     Input
		repo      *userRepoMock
		hasher    *hasherMock
		wantErr   error
		wantOut   Output
		wantEmail string
	}{
		{
			name:      "success normalizes email",
			input:     Input{Email: "  Alice@Example.COM ", Password: "password1", Name: "Alice"},
			repo:      &userRepoMock{createID: 42},
			hasher:    &hasherMock{hash: "hashed"},
			wantOut:   Output{UserID: 42, Email: "alice@example.com", Name: "Alice"},
			wantEmail: "alice@example.com",
		},
		{
			name:    "invalid email",
			input:   Input{Email: "nope", Password: "password1", Name: "Alice"},
			repo:    &userRepoMock{},
			hasher:  &hasherMock{hash: "hashed"},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "short password",
			input:   Input{Email: "a@b.com", Password: "short", Name: "Alice"},
			repo:    &userRepoMock{},
			hasher:  &hasherMock{hash: "hashed"},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "duplicate email",
			input:   Input{Email: "a@b.com", Password: "password1", Name: "Alice"},
			repo:    &userRepoMock{createErr: domain.ErrAlreadyExists},
			hasher:  &hasherMock{hash: "hashed"},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name:    "hasher failure",
			input:   Input{Email: "a@b.com", Password: "password1", Name: "Alice"},
			repo:    &userRepoMock{},
			hasher:  &hasherMock{err: errors.New("boom")},
			wantErr: nil, // generic error, checked separately below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.repo, tt.hasher)

			out, err := uc.Handle(context.Background(), tt.input)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.hasher.err != nil:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.wantOut, out)
				assert.Equal(t, tt.wantEmail, tt.repo.gotUser.Email)
				assert.Equal(t, "hashed", tt.repo.gotUser.PasswordHash)
			}
		})
	}
}

func TestUsecase_Handle_DuplicateEmailMessage(t *testing.T) {
	uc := New(&userRepoMock{createErr: domain.ErrAlreadyExists}, &hasherMock{hash: "hashed"})

	_, err := uc.Handle(context.Background(), Input{Email: "a@b.com", Password: "password1", Name: "Alice"})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
	var safeErr *domain.SafeError
	require.ErrorAs(t, err, &safeErr, "the client must receive a safe, human-readable message")
	assert.Equal(t, "user with this email already exists", safeErr.Msg)
}
