package http

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/lib/authctx"
)

// rateLimiter decides whether a request identified by key may proceed.
type rateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration, err error)
}

// NewRateLimitMiddleware limits requests per authenticated user, falling back
// to the client IP for anonymous endpoints. Limiter outages fail open: the
// request proceeds and the incident is logged.
func NewRateLimitMiddleware(limiter rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			allowed, retryAfter, err := limiter.Allow(ctx, limiterKey(r))
			if err != nil {
				slog.WarnContext(ctx, "rate limiter unavailable, failing open", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
				respondError(ctx, w, fmt.Errorf("rate limit exceeded: %w",
					&domain.SafeError{Kind: domain.ErrRateLimited, Msg: "rate limit exceeded, retry later"}))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func limiterKey(r *http.Request) string {
	if userID, ok := authctx.UserID(r.Context()); ok {
		return "user:" + strconv.FormatInt(userID, 10)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}
