// Package taskcache caches per-team task listing pages in Redis.
//
// Invalidation uses versioned keys: every team has a version counter and all
// page keys embed the current version. Bumping the counter on any task
// mutation makes older keys unreachable; they expire via TTL. This avoids
// SCAN-based deletion entirely.
package taskcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"team-taskflow/internal/domain"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{client: client, ttl: ttl}
}

type cachedTask struct {
	ID          int64     `json:"id"`
	TeamID      int64     `json:"team_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	AssigneeID  *int64    `json:"assignee_id,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type cachedPage struct {
	Tasks []cachedTask `json:"tasks"`
	Total int64        `json:"total"`
}

// Get returns the cached page for the filter, with a hit flag.
func (c *Cache) Get(ctx context.Context, filter domain.TaskFilter) (domain.TaskPage, bool, error) {
	key, err := c.pageKey(ctx, filter)
	if err != nil {
		return domain.TaskPage{}, false, err
	}

	payload, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.TaskPage{}, false, nil
		}
		return domain.TaskPage{}, false, fmt.Errorf("reading cached page: %w", err)
	}

	var page cachedPage
	if err := json.Unmarshal(payload, &page); err != nil {
		return domain.TaskPage{}, false, fmt.Errorf("decoding cached page: %w", err)
	}
	return page.toDomain(), true, nil
}

// Set stores the page under the team's current cache version with a TTL.
func (c *Cache) Set(ctx context.Context, filter domain.TaskFilter, page domain.TaskPage) error {
	key, err := c.pageKey(ctx, filter)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(toCached(page))
	if err != nil {
		return fmt.Errorf("encoding page for cache: %w", err)
	}

	if err := c.client.Set(ctx, key, payload, c.ttl).Err(); err != nil {
		return fmt.Errorf("writing cached page: %w", err)
	}
	return nil
}

// InvalidateTeam bumps the team's cache version, orphaning all cached pages.
func (c *Cache) InvalidateTeam(ctx context.Context, teamID int64) error {
	if err := c.client.Incr(ctx, versionKey(teamID)).Err(); err != nil {
		return fmt.Errorf("bumping team cache version: %w", err)
	}
	return nil
}

func (c *Cache) pageKey(ctx context.Context, filter domain.TaskFilter) (string, error) {
	version, err := c.client.Get(ctx, versionKey(filter.TeamID)).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("reading team cache version: %w", err)
	}

	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	assignee := ""
	if filter.AssigneeID != nil {
		assignee = strconv.FormatInt(*filter.AssigneeID, 10)
	}

	return fmt.Sprintf("tasks:team:%d:v%d:s=%s:a=%s:p=%d:ps=%d",
		filter.TeamID, version, status, assignee, filter.Page, filter.PageSize), nil
}

func versionKey(teamID int64) string {
	return fmt.Sprintf("tasks:team:%d:ver", teamID)
}

func toCached(page domain.TaskPage) cachedPage {
	tasks := make([]cachedTask, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		tasks = append(tasks, cachedTask{
			ID:          task.ID,
			TeamID:      task.TeamID,
			Title:       task.Title,
			Description: task.Description,
			Status:      string(task.Status),
			AssigneeID:  task.AssigneeID,
			CreatedBy:   task.CreatedBy,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		})
	}
	return cachedPage{Tasks: tasks, Total: page.Total}
}

func (p cachedPage) toDomain() domain.TaskPage {
	tasks := make([]domain.Task, 0, len(p.Tasks))
	for _, task := range p.Tasks {
		tasks = append(tasks, domain.Task{
			ID:          task.ID,
			TeamID:      task.TeamID,
			Title:       task.Title,
			Description: task.Description,
			Status:      domain.TaskStatus(task.Status),
			AssigneeID:  task.AssigneeID,
			CreatedBy:   task.CreatedBy,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		})
	}
	return domain.TaskPage{Tasks: tasks, Total: p.Total}
}
