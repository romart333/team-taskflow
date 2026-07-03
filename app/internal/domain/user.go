package domain

import (
	"fmt"
	"net/mail"
	"time"
)

// MinPasswordLength is the domain rule for the shortest acceptable password.
const MinPasswordLength = 8

type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ValidateRegistration checks the invariants for a new account.
func ValidateRegistration(email, password, name string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return NewValidationError("email is not a valid address")
	}
	if len(password) < MinPasswordLength {
		return NewValidationError(fmt.Sprintf("password must be at least %d characters", MinPasswordLength))
	}
	if name == "" {
		return NewValidationError("name is required")
	}
	return nil
}
