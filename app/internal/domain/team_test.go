package domain

import (
	"strings"
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

func TestValidateTeamName(t *testing.T) {
	assert.NoError(t, ValidateTeamName("Platform"))
	assert.ErrorIs(t, ValidateTeamName(""), ErrValidation)
	assert.NoError(t, ValidateTeamName(strings.Repeat("n", MaxTeamNameLength)))
	assert.ErrorIs(t, ValidateTeamName(strings.Repeat("n", MaxTeamNameLength+1)), ErrValidation)
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
