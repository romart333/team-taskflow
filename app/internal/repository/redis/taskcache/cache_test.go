package taskcache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

func newTestCache(t *testing.T, ttl time.Duration) (*Cache, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewCache(client, ttl), mr
}

func samplePage() domain.TaskPage {
	assignee := int64(9)
	completedAt := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	return domain.TaskPage{
		Tasks: []domain.Task{
			{
				ID:          1,
				TeamID:      7,
				Title:       "Fix bug",
				Description: "details",
				Status:      domain.TaskStatusDone,
				AssigneeID:  &assignee,
				CompletedAt: &completedAt,
				CreatedBy:   5,
				CreatedAt:   time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			},
			{ID: 2, TeamID: 7, Title: "Write docs", Status: domain.TaskStatusTodo, CreatedBy: 5},
		},
		Total: 42,
	}
}

func TestCache_GetMissThenSetThenHit(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	_, hit, version, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	require.False(t, hit)

	require.NoError(t, cache.Set(ctx, filter, version, samplePage()))

	got, hit, _, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	require.True(t, hit)
	assert.Equal(t, samplePage(), got, "page must survive the cache round-trip unchanged")
}

func TestCache_FilterIsPartOfTheKey(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx := context.Background()
	statusDone := domain.TaskStatusDone
	assignee := int64(9)

	base := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}
	_, _, version, err := cache.Get(ctx, base)
	require.NoError(t, err)
	require.NoError(t, cache.Set(ctx, base, version, samplePage()))

	variants := []domain.TaskFilter{
		{TeamID: 7, Page: 2, PageSize: 20},
		{TeamID: 7, Page: 1, PageSize: 50},
		{TeamID: 7, Page: 1, PageSize: 20, Status: &statusDone},
		{TeamID: 7, Page: 1, PageSize: 20, AssigneeID: &assignee},
		{TeamID: 8, Page: 1, PageSize: 20},
	}
	for _, filter := range variants {
		_, hit, _, err := cache.Get(ctx, filter)
		require.NoError(t, err)
		assert.False(t, hit, "filter %+v must not share a key with the base filter", filter)
	}
}

func TestCache_InvalidateTeamOrphansCachedPages(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	_, _, version, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	require.NoError(t, cache.Set(ctx, filter, version, samplePage()))

	require.NoError(t, cache.InvalidateTeam(ctx, filter.TeamID))

	_, hit, newVersion, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	assert.False(t, hit, "pages cached under the old version must be unreachable")
	assert.Equal(t, version+1, newVersion)
}

func TestCache_InvalidateTeamIsScopedToTeam(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	_, _, version, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	require.NoError(t, cache.Set(ctx, filter, version, samplePage()))

	require.NoError(t, cache.InvalidateTeam(ctx, 8))

	_, hit, _, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	assert.True(t, hit, "invalidating another team must not touch this team's pages")
}

// A page read from the DB before an invalidation bump must be stored under the
// stale version the caller observed, staying unreachable for readers.
func TestCache_SetWithStaleVersionDoesNotPoisonCache(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	_, _, staleVersion, err := cache.Get(ctx, filter)
	require.NoError(t, err)

	require.NoError(t, cache.InvalidateTeam(ctx, filter.TeamID))

	require.NoError(t, cache.Set(ctx, filter, staleVersion, samplePage()))

	_, hit, _, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	assert.False(t, hit, "a write under a pre-bump version must not be visible")
}

func TestCache_EntriesExpireByTTL(t *testing.T) {
	cache, mr := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	_, _, version, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	require.NoError(t, cache.Set(ctx, filter, version, samplePage()))

	mr.FastForward(time.Minute + time.Second)

	_, hit, _, err := cache.Get(ctx, filter)
	require.NoError(t, err)
	assert.False(t, hit, "entries must expire after the TTL")
}

func TestCache_CorruptedPayload(t *testing.T) {
	cache, mr := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	require.NoError(t, mr.Set(pageKey(filter, 0), "not-json"))

	_, _, _, err := cache.Get(ctx, filter)
	require.Error(t, err)
}

func TestCache_RedisFailure(t *testing.T) {
	cache, mr := newTestCache(t, time.Minute)
	ctx := context.Background()
	filter := domain.TaskFilter{TeamID: 7, Page: 1, PageSize: 20}

	mr.Close()

	_, _, _, err := cache.Get(ctx, filter)
	require.Error(t, err)

	require.Error(t, cache.Set(ctx, filter, 0, samplePage()))
	require.Error(t, cache.InvalidateTeam(ctx, filter.TeamID))
}
