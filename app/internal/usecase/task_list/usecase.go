package task_list

import (
	"context"
	"fmt"
	"log/slog"
)

type Usecase struct {
	tasks      TaskRepository
	access     TeamAccess
	cache      TaskListCache
	pagination Pagination
}

func New(tasks TaskRepository, access TeamAccess, cache TaskListCache, pagination Pagination) *Usecase {
	return &Usecase{tasks: tasks, access: access, cache: cache, pagination: pagination}
}

// Handle returns a filtered, paginated task listing for a team the actor
// belongs to. Listings are served from cache when possible; cache failures
// degrade to the database and never fail the request.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	filter := in.Filter.Normalize(u.pagination.DefaultPageSize, u.pagination.MaxPageSize)

	if err := u.access.EnsureTeamMember(ctx, filter.TeamID, in.ActorID); err != nil {
		return Output{}, fmt.Errorf("authorizing actor: %w", err)
	}

	page, hit, version, err := u.cache.Get(ctx, filter)
	cacheUsable := err == nil
	if err != nil {
		slog.WarnContext(ctx, "task list cache read failed", "team_id", filter.TeamID, "error", err)
	}
	if hit {
		return Output{Page: page, PageNum: filter.Page, PageSize: filter.PageSize}, nil
	}

	page, err = u.tasks.List(ctx, filter)
	if err != nil {
		return Output{}, fmt.Errorf("listing tasks: %w", err)
	}

	// Without a version observed before the DB read the write cannot be made
	// invalidation-safe, so a failed cache read also skips the write.
	if cacheUsable {
		if err := u.cache.Set(ctx, filter, version, page); err != nil {
			slog.WarnContext(ctx, "task list cache write failed", "team_id", filter.TeamID, "error", err)
		}
	}

	return Output{Page: page, PageNum: filter.Page, PageSize: filter.PageSize}, nil
}
