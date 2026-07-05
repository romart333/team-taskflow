package task_list

import (
	"context"

	"team-taskflow/internal/domain"
)

type Input struct {
	ActorID int64
	Filter  domain.TaskFilter
}

type Output struct {
	Page     domain.TaskPage
	PageNum  int
	PageSize int
}

// Pagination carries page-size limits injected from configuration.
type Pagination struct {
	DefaultPageSize int
	MaxPageSize     int
}

// TaskRepository is the read port for task listings.
type TaskRepository interface {
	List(ctx context.Context, filter domain.TaskFilter) (domain.TaskPage, error)
}

// TeamRepository checks team memberships.
type TeamRepository interface {
	// GetMember returns domain.ErrNotFound when the user is not a member.
	GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error)
}

// TaskListCache caches task listing pages per team (best effort).
type TaskListCache interface {
	// Get returns the cached page, a hit flag and the cache version observed
	// at read time. Pass that version to Set: a concurrent invalidation bumps
	// the version, making a stale write land on an unreachable key instead of
	// re-poisoning the cache.
	Get(ctx context.Context, filter domain.TaskFilter) (page domain.TaskPage, hit bool, version int64, err error)
	Set(ctx context.Context, filter domain.TaskFilter, version int64, page domain.TaskPage) error
}
