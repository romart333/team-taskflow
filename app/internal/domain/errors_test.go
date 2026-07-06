package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
		msg      string
	}{
		{"validation", NewValidationError("bad input"), ErrValidation, "bad input"},
		{"not found", NewNotFoundError("no such team"), ErrNotFound, "no such team"},
		{"conflict", NewConflictError("already member"), ErrConflict, "already member"},
		{"permission", NewPermissionDeniedError("not allowed"), ErrPermissionDenied, "not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.err, tt.sentinel)
			assert.Equal(t, tt.msg, tt.err.Error())

			wrapped := fmt.Errorf("context: %w", tt.err)
			require.ErrorIs(t, wrapped, tt.sentinel)

			safe, ok := errors.AsType[*SafeError](wrapped)
			require.True(t, ok)
			assert.Equal(t, tt.msg, safe.Msg)
		})
	}
}

func TestValidateHelpers(t *testing.T) {
	assert.NoError(t, ValidateTeamName("Platform"))
	assert.ErrorIs(t, ValidateTeamName(""), ErrValidation)
	assert.NoError(t, ValidateCommentBody("hi"))
	assert.ErrorIs(t, ValidateCommentBody(""), ErrValidation)
}
