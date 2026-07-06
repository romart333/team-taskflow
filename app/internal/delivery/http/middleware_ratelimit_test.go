package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"team-taskflow/internal/lib/authctx"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allowed request passes through with user key", func(t *testing.T) {
		limiter := newMockrateLimiter(t)
		limiter.EXPECT().Allow(mock.Anything, "user:42").Return(true, time.Duration(0), nil)
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(authctx.WithUserID(req.Context(), 42))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("anonymous request is keyed by IP", func(t *testing.T) {
		limiter := newMockrateLimiter(t)
		limiter.EXPECT().Allow(mock.Anything, "ip:192.0.2.10").Return(true, time.Duration(0), nil)
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:51234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("denied request gets 429 with Retry-After", func(t *testing.T) {
		limiter := newMockrateLimiter(t)
		limiter.EXPECT().Allow(mock.Anything, mock.Anything).Return(false, 42*time.Second, nil)
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "42", rec.Header().Get("Retry-After"))
		assert.Contains(t, rec.Body.String(), "rate limit exceeded")
	})

	t.Run("limiter outage fails open", func(t *testing.T) {
		limiter := newMockrateLimiter(t)
		limiter.EXPECT().Allow(mock.Anything, mock.Anything).Return(false, time.Duration(0), errors.New("redis down"))
		handler := NewRateLimitMiddleware(limiter)(okHandler())

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
