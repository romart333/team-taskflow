package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MinPasswordLength is the domain rule for the shortest acceptable password.
	MinPasswordLength = 8
	// MaxEmailLength mirrors the users.email VARCHAR(255) column.
	MaxEmailLength = 255
)

type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NormalizeEmail canonicalizes a raw email address: it trims whitespace,
// rejects RFC 5322 name-addr forms (only a bare address may identify an
// account) and lowercases the result, so the DB uniqueness constraint
// guarantees one account per mailbox. It is the single owner of email
// normalization for registration, login and invites.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", NewValidationError("email must be a plain valid address")
	}
	if utf8.RuneCountInString(trimmed) > MaxEmailLength {
		return "", NewValidationError(fmt.Sprintf("email must be at most %d characters", MaxEmailLength))
	}
	return strings.ToLower(addr.Address), nil
}

// ValidateRegistration checks the invariants for a new account. The email is
// validated and normalized separately via NormalizeEmail.
func ValidateRegistration(password, name string) error {
	if len(password) < MinPasswordLength {
		return NewValidationError(fmt.Sprintf("password must be at least %d characters", MinPasswordLength))
	}
	if name == "" {
		return NewValidationError("name is required")
	}
	return nil
}
