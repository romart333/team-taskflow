package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"team-taskflow/internal/lib/authctx"
)

type limiterMock struct {
	allowed    bool
	retryAfter time.Duration
	err        error
	gotKey     string
}

func (m *limiterMock) Allow(_ context.Context, key string) (bool, time.Duration, error) {
	m.gotKey = key
	return m.allowed, m.retryAfter, m.err
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allowed request passes through with user key", func(t *testing.T) {
		limiter := &limiterMock{allowed: true}
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(authctx.WithUserID(req.Context(), 42))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "user:42", limiter.gotKey)
	})

	t.Run("anonymous request is keyed by IP", func(t *testing.T) {
		limiter := &limiterMock{allowed: true}
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:51234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ip:192.0.2.10", limiter.gotKey)
	})

	t.Run("denied request gets 429 with Retry-After", func(t *testing.T) {
		limiter := &limiterMock{allowed: false, retryAfter: 42 * time.Second}
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "42", rec.Header().Get("Retry-After"))
		assert.Contains(t, rec.Body.String(), "rate limit exceeded")
	})

	t.Run("limiter outage fails open", func(t *testing.T) {
		limiter := &limiterMock{err: errors.New("redis down")}
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
