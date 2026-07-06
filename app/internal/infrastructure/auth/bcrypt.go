package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher implements the usecase PasswordHasher/PasswordVerifier ports
// on top of bcrypt.
type PasswordHasher struct {
	cost int
}

func NewPasswordHasher(cost int) (*PasswordHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost %d out of range [%d, %d]", cost, bcrypt.MinCost, bcrypt.MaxCost)
	}
	return &PasswordHasher{cost: cost}, nil
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("generating bcrypt hash: %w", err)
	}
	return string(hash), nil
}

func (h *PasswordHasher) Verify(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("comparing bcrypt hash: %w", err)
	}
	return nil
}
