package commentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/infrastructure/tx"
)

type Repository struct {
	pool *sql.DB
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, comment domain.TaskComment) (int64, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`INSERT INTO task_comments (task_id, user_id, body) VALUES (?, ?, ?)`,
		comment.TaskID, comment.UserID, comment.Body,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting comment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading inserted comment id: %w", err)
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, commentID int64) (domain.TaskComment, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	var entity commentEntity
	err := executor.QueryRowContext(ctx,
		`SELECT id, task_id, user_id, body, created_at FROM task_comments WHERE id = ?`, commentID,
	).Scan(&entity.ID, &entity.TaskID, &entity.UserID, &entity.Body, &entity.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskComment{}, fmt.Errorf("no comment with id=%d: %w", commentID, domain.ErrNotFound)
		}
		return domain.TaskComment{}, fmt.Errorf("selecting comment: %w", err)
	}
	return entity.toDomain(), nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID int64) ([]domain.TaskComment, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	rows, err := executor.QueryContext(ctx,
		`SELECT id, task_id, user_id, body, created_at
		 FROM task_comments WHERE task_id = ? ORDER BY created_at, id`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("selecting comments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var comments []domain.TaskComment
	for rows.Next() {
		var entity commentEntity
		if err := rows.Scan(&entity.ID, &entity.TaskID, &entity.UserID, &entity.Body, &entity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning comment row: %w", err)
		}
		comments = append(comments, entity.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating comment rows: %w", err)
	}
	return comments, nil
}
