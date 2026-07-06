package auth_login

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
	dbErr := errors.New("db down")

	tests := []struct {
		name        string
		setup       func(repo *MockUserRepository, verifier *MockPasswordVerifier, issuer *MockTokenIssuer)
		wantErr     error
		wantRepoErr bool
		want        Output
	}{
		{
			name: "success",
			setup: func(repo *MockUserRepository, verifier *MockPasswordVerifier, issuer *MockTokenIssuer) {
				repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).
					Return(domain.User{ID: 1, PasswordHash: "h"}, nil)
				verifier.EXPECT().Verify("h", "pw").Return(nil)
				issuer.EXPECT().Issue(int64(1)).Return("jwt-token", nil)
			},
			want: Output{AccessToken: "jwt-token"},
		},
		{
			name: "unknown email maps to unauthorized",
			setup: func(repo *MockUserRepository, verifier *MockPasswordVerifier, issuer *MockTokenIssuer) {
				repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).
					Return(domain.User{}, domain.ErrNotFound)
			},
			wantErr: domain.ErrUnauthorized,
		},
		{
			name: "wrong password maps to unauthorized",
			setup: func(repo *MockUserRepository, verifier *MockPasswordVerifier, issuer *MockTokenIssuer) {
				repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).
					Return(domain.User{ID: 1, PasswordHash: "h"}, nil)
				verifier.EXPECT().Verify("h", "pw").Return(errors.New("mismatch"))
			},
			wantErr: domain.ErrUnauthorized,
		},
		{
			name: "repository failure is not unauthorized",
			setup: func(repo *MockUserRepository, verifier *MockPasswordVerifier, issuer *MockTokenIssuer) {
				repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).
					Return(domain.User{}, dbErr)
			},
			wantRepoErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepository(t)
			verifier := NewMockPasswordVerifier(t)
			issuer := NewMockTokenIssuer(t)
			tt.setup(repo, verifier, issuer)
			uc := New(repo, verifier, issuer)

			out, err := uc.Handle(context.Background(), Input{Email: "A@b.com", Password: "pw"})

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantRepoErr:
				require.Error(t, err)
				assert.NotErrorIs(t, err, domain.ErrUnauthorized)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.want, out)
			}
		})
	}
}
