// Package ratelimit implements a Redis-backed fixed-window rate limiter.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter counts requests per key in fixed windows stored in Redis, so the
// limit is enforced consistently across service replicas.
type Limiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	now    func() time.Time
}

func NewLimiter(client *redis.Client, limit int, window time.Duration) *Limiter {
	return &Limiter{client: client, limit: limit, window: window, now: time.Now}
}

// Allow reports whether the request identified by key fits the limit. When
// denied, it returns how long the caller should wait before retrying.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	now := l.now()
	windowStart := now.Truncate(l.window)
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, windowStart.Unix())

	pipe := l.client.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("incrementing rate limit counter: %w", err)
	}

	if incr.Val() > int64(l.limit) {
		return false, windowStart.Add(l.window).Sub(now), nil
	}
	return true, 0, nil
}
