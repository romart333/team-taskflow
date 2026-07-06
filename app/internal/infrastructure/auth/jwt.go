package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"team-taskflow/internal/domain"
)

// JWTManager issues and parses HS256 access tokens. It implements the
// usecase TokenIssuer port and backs the authentication middleware.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (m *JWTManager) Issue(userID int64) (string, error) {
	now := m.now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

// Parse validates the token and returns the authenticated user ID.
func (m *JWTManager) Parse(token string) (int64, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return 0, fmt.Errorf("parsing token: %w", domain.ErrUnauthorized)
	}

	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, fmt.Errorf("unexpected claims type: %w", domain.ErrUnauthorized)
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing subject %q: %w", claims.Subject, domain.ErrUnauthorized)
	}
	return userID, nil
}
