package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLimiter(t *testing.T, limit int, window time.Duration, now time.Time) *Limiter {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	limiter := NewLimiter(client, limit, window)
	limiter.now = func() time.Time { return now }
	return limiter
}

func TestLimiter_Allow_EnforcesLimitWithinWindow(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 30, 0, time.UTC)
	limiter := newTestLimiter(t, 3, time.Minute, now)
	ctx := context.Background()

	for i := range 3 {
		allowed, retryAfter, err := limiter.Allow(ctx, "user:1")
		require.NoError(t, err, "request %d", i+1)
		assert.True(t, allowed, "request %d must fit the limit", i+1)
		assert.Zero(t, retryAfter)
	}

	allowed, retryAfter, err := limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	assert.False(t, allowed)
	// The window started at 10:00:00, so at 10:00:30 the next one opens in 30s.
	assert.Equal(t, 30*time.Second, retryAfter)
}

func TestLimiter_Allow_KeysAreIndependent(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	limiter := newTestLimiter(t, 1, time.Minute, now)
	ctx := context.Background()

	allowed, _, err := limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, _, err = limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	assert.False(t, allowed, "second request for the same key must be denied")

	allowed, _, err = limiter.Allow(ctx, "user:2")
	require.NoError(t, err)
	assert.True(t, allowed, "another key must have its own counter")
}

func TestLimiter_Allow_NewWindowResetsCounter(t *testing.T) {
	start := time.Date(2026, 1, 15, 10, 0, 59, 0, time.UTC)
	limiter := newTestLimiter(t, 1, time.Minute, start)
	ctx := context.Background()

	allowed, _, err := limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, _, err = limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	require.False(t, allowed)

	limiter.now = func() time.Time { return start.Add(time.Second) }

	allowed, _, err = limiter.Allow(ctx, "user:1")
	require.NoError(t, err)
	assert.True(t, allowed, "next fixed window must start with a fresh counter")
}

func TestLimiter_Allow_RedisFailure(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	limiter := NewLimiter(client, 1, time.Minute)

	mr.Close()

	allowed, _, err := limiter.Allow(context.Background(), "user:1")
	require.Error(t, err)
	assert.False(t, allowed)
}
