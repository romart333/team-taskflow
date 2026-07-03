package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_CanInvite(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.CanInvite())
		})
	}
}

func TestParseRole(t *testing.T) {
	for _, valid := range []string{"owner", "admin", "member"} {
		role, err := ParseRole(valid)
		require.NoError(t, err)
		assert.Equal(t, Role(valid), role)
	}

	_, err := ParseRole("superuser")
	require.ErrorIs(t, err, ErrValidation)
}

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		pass    string
		user    string
		wantErr bool
	}{
		{"valid", "a@b.com", "password1", "Alice", false},
		{"bad email", "not-an-email", "password1", "Alice", true},
		{"short password", "a@b.com", "short", "Alice", true},
		{"empty name", "a@b.com", "password1", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistration(tt.email, tt.pass, tt.user)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrValidation)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
