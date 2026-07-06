package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	t.Run("trims and lowercases a bare address", func(t *testing.T) {
		email, err := NormalizeEmail("  BOB@X.com ")
		require.NoError(t, err)
		assert.Equal(t, "bob@x.com", email)
	})

	t.Run("rejects RFC 5322 name-addr forms", func(t *testing.T) {
		for _, raw := range []string{"Bob <bob@x.com>", "<bob@x.com>", `"Bob" <bob@x.com>`} {
			_, err := NormalizeEmail(raw)
			require.ErrorIs(t, err, ErrValidation, "name-addr form %q must be rejected", raw)
		}
	})

	t.Run("rejects invalid addresses", func(t *testing.T) {
		for _, raw := range []string{"", "not-an-email", "a@", "@x.com", "a b@x.com"} {
			_, err := NormalizeEmail(raw)
			require.ErrorIs(t, err, ErrValidation, "invalid address %q must be rejected", raw)
		}
	})

	t.Run("rejects addresses longer than the column limit", func(t *testing.T) {
		local := strings.Repeat("a", MaxEmailLength)
		_, err := NormalizeEmail(local + "@x.com")
		require.ErrorIs(t, err, ErrValidation)
	})
}

func TestValidateRegistration(t *testing.T) {
	assert.NoError(t, ValidateRegistration("password8", "Alice"))
	assert.ErrorIs(t, ValidateRegistration("short", "Alice"), ErrValidation)
	assert.ErrorIs(t, ValidateRegistration("password8", ""), ErrValidation)

	t.Run("password length is capped by the bcrypt input limit", func(t *testing.T) {
		assert.NoError(t, ValidateRegistration(strings.Repeat("p", MaxPasswordLength), "Alice"))
		assert.ErrorIs(t, ValidateRegistration(strings.Repeat("p", MaxPasswordLength+1), "Alice"), ErrValidation)
	})

	t.Run("name length is capped by the column limit", func(t *testing.T) {
		assert.NoError(t, ValidateRegistration("password8", strings.Repeat("n", MaxUserNameLength)))
		assert.ErrorIs(t, ValidateRegistration("password8", strings.Repeat("n", MaxUserNameLength+1)), ErrValidation)
	})
}
