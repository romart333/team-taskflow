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

// TeamAccess authorizes the actor as a member of a team.
type TeamAccess interface {
	// EnsureTeamMember returns a client-visible domain.ErrPermissionDenied
	// when the actor is not a member of the team.
	EnsureTeamMember(ctx context.Context, teamID, actorID int64) error
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
