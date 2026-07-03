package taskrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/infrastructure/tx"
)

const taskColumns = "id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at"

type Repository struct {
	pool *sql.DB
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task domain.Task) (int64, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`INSERT INTO tasks (team_id, title, description, status, assignee_id, created_by)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		task.TeamID, task.Title, task.Description, string(task.Status),
		assigneeParam(task.AssigneeID), task.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading inserted task id: %w", err)
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, taskID int64) (domain.Task, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	var entity taskEntity
	err := executor.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID,
	).Scan(&entity.ID, &entity.TeamID, &entity.Title, &entity.Description, &entity.Status,
		&entity.AssigneeID, &entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, fmt.Errorf("no task with id=%d: %w", taskID, domain.ErrNotFound)
		}
		return domain.Task{}, fmt.Errorf("selecting task: %w", err)
	}
	return entity.toDomain(), nil
}

func (r *Repository) Update(ctx context.Context, task domain.Task) error {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, assignee_id = ? WHERE id = ?`,
		task.Title, task.Description, string(task.Status), assigneeParam(task.AssigneeID), task.ID,
	)
	if err != nil {
		return fmt.Errorf("updating task id=%d: %w", task.ID, err)
	}

	// The pool runs with CLIENT_FOUND_ROWS, so RowsAffected counts matched
	// rows and 0 unambiguously means the task does not exist.
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no task with id=%d: %w", task.ID, domain.ErrNotFound)
	}
	return nil
}

// List returns one page of team tasks plus the total count for the filter.
// Pagination happens in the database via LIMIT/OFFSET.
func (r *Repository) List(ctx context.Context, filter domain.TaskFilter) (domain.TaskPage, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	where := []string{"team_id = ?"}
	args := []any{filter.TeamID}
	if filter.Status != nil {
		where = append(where, "status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.AssigneeID != nil {
		where = append(where, "assignee_id = ?")
		args = append(args, *filter.AssigneeID)
	}
	condition := strings.Join(where, " AND ")

	var total int64
	if err := executor.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tasks WHERE `+condition, args...,
	).Scan(&total); err != nil {
		return domain.TaskPage{}, fmt.Errorf("counting tasks: %w", err)
	}

	rows, err := executor.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE `+condition+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, filter.PageSize, filter.Offset())...,
	)
	if err != nil {
		return domain.TaskPage{}, fmt.Errorf("selecting tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]domain.Task, 0, filter.PageSize)
	for rows.Next() {
		var entity taskEntity
		if err := rows.Scan(&entity.ID, &entity.TeamID, &entity.Title, &entity.Description, &entity.Status,
			&entity.AssigneeID, &entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt); err != nil {
			return domain.TaskPage{}, fmt.Errorf("scanning task row: %w", err)
		}
		tasks = append(tasks, entity.toDomain())
	}
	if err := rows.Err(); err != nil {
		return domain.TaskPage{}, fmt.Errorf("iterating task rows: %w", err)
	}

	return domain.TaskPage{Tasks: tasks, Total: total}, nil
}
