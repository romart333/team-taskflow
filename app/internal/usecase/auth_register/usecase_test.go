package auth_register

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

func TestUsecase_Handle(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		setup       func(repo *MockUserRepository, hasher *MockPasswordHasher)
		wantErr     error
		wantHashErr bool
		wantOut     Output
	}{
		{
			name:  "success normalizes email",
			input: Input{Email: "  Alice@Example.COM ", Password: "password1", Name: "Alice"},
			setup: func(repo *MockUserRepository, hasher *MockPasswordHasher) {
				hasher.EXPECT().Hash("password1").Return("hashed", nil)
				repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(user domain.User) bool {
					return user.Email == "alice@example.com" && user.PasswordHash == "hashed"
				})).Return(42, nil)
			},
			wantOut: Output{UserID: 42, Email: "alice@example.com", Name: "Alice"},
		},
		{
			name:    "invalid email",
			input:   Input{Email: "nope", Password: "password1", Name: "Alice"},
			wantErr: domain.ErrValidation,
		},
		{
			name:    "short password",
			input:   Input{Email: "a@b.com", Password: "short", Name: "Alice"},
			wantErr: domain.ErrValidation,
		},
		{
			name:  "duplicate email",
			input: Input{Email: "a@b.com", Password: "password1", Name: "Alice"},
			setup: func(repo *MockUserRepository, hasher *MockPasswordHasher) {
				hasher.EXPECT().Hash("password1").Return("hashed", nil)
				repo.EXPECT().Create(mock.Anything, mock.Anything).Return(0, domain.ErrAlreadyExists)
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name:  "hasher failure",
			input: Input{Email: "a@b.com", Password: "password1", Name: "Alice"},
			setup: func(repo *MockUserRepository, hasher *MockPasswordHasher) {
				hasher.EXPECT().Hash("password1").Return("", errors.New("boom"))
			},
			wantHashErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepository(t)
			hasher := NewMockPasswordHasher(t)
			if tt.setup != nil {
				tt.setup(repo, hasher)
			}
			uc := New(repo, hasher)

			out, err := uc.Handle(context.Background(), tt.input)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantHashErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.wantOut, out)
			}
		})
	}
}

func TestUsecase_Handle_DuplicateEmailMessage(t *testing.T) {
	repo := NewMockUserRepository(t)
	hasher := NewMockPasswordHasher(t)
	hasher.EXPECT().Hash("password1").Return("hashed", nil)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(0, domain.ErrAlreadyExists)
	uc := New(repo, hasher)

	_, err := uc.Handle(context.Background(), Input{Email: "a@b.com", Password: "password1", Name: "Alice"})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
	var safeErr *domain.SafeError
	require.ErrorAs(t, err, &safeErr, "the client must receive a safe, human-readable message")
	assert.Equal(t, "user with this email already exists", safeErr.Msg)
}
