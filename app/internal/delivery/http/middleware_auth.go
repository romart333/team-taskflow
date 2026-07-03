package http

import (
	"fmt"
	"net/http"
	"strings"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/lib/authctx"
)

const bearerPrefix = "Bearer "

// tokenParser validates an access token and returns the user ID it carries.
type tokenParser interface {
	Parse(token string) (int64, error)
}

// NewAuthMiddleware authenticates requests via the Authorization header and
// stores the user ID in the request context.
func NewAuthMiddleware(parser tokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				respondError(ctx, w, fmt.Errorf("missing bearer token: %w", domain.ErrUnauthorized))
				return
			}

			userID, err := parser.Parse(strings.TrimPrefix(header, bearerPrefix))
			if err != nil {
				respondError(ctx, w, fmt.Errorf("authenticating request: %w", err))
				return
			}

			next.ServeHTTP(w, r.WithContext(authctx.WithUserID(ctx, userID)))
		})
	}
}

// actorID extracts the authenticated user ID placed by the auth middleware.
func actorID(r *http.Request) (int64, error) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		return 0, fmt.Errorf("no authenticated user in context: %w", domain.ErrUnauthorized)
	}
	return userID, nil
}
