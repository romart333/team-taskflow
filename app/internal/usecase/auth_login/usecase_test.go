package auth_login

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

type userRepoMock struct {
	user domain.User
	err  error
}

func (m *userRepoMock) GetByEmail(context.Context, string) (domain.User, error) {
	return m.user, m.err
}

type verifierMock struct{ err error }

func (m *verifierMock) Verify(string, string) error { return m.err }

type issuerMock struct {
	token string
	err   error
}

func (m *issuerMock) Issue(int64) (string, error) { return m.token, m.err }

func TestUsecase_Handle(t *testing.T) {
	tests := []struct {
		name     string
		repo     *userRepoMock
		verifier *verifierMock
		issuer   *issuerMock
		wantErr  error
		want     Output
	}{
		{
			name:     "success",
			repo:     &userRepoMock{user: domain.User{ID: 1, PasswordHash: "h"}},
			verifier: &verifierMock{},
			issuer:   &issuerMock{token: "jwt-token"},
			want:     Output{AccessToken: "jwt-token"},
		},
		{
			name:     "unknown email maps to unauthorized",
			repo:     &userRepoMock{err: domain.ErrNotFound},
			verifier: &verifierMock{},
			issuer:   &issuerMock{},
			wantErr:  domain.ErrUnauthorized,
		},
		{
			name:     "wrong password maps to unauthorized",
			repo:     &userRepoMock{user: domain.User{ID: 1, PasswordHash: "h"}},
			verifier: &verifierMock{err: errors.New("mismatch")},
			issuer:   &issuerMock{},
			wantErr:  domain.ErrUnauthorized,
		},
		{
			name:     "repository failure is not unauthorized",
			repo:     &userRepoMock{err: errors.New("db down")},
			verifier: &verifierMock{},
			issuer:   &issuerMock{},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.repo, tt.verifier, tt.issuer)

			out, err := uc.Handle(context.Background(), Input{Email: "A@b.com", Password: "pw"})

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.repo.err != nil:
				require.Error(t, err)
				assert.NotErrorIs(t, err, domain.ErrUnauthorized)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.want, out)
			}
		})
	}
}
