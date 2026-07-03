package historyrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/infrastructure/tx"
)

type Repository struct {
	pool *sql.DB
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool}
}

// AddEntries inserts audit entries in one statement. It is called inside the
// task update transaction.
func (r *Repository) AddEntries(ctx context.Context, entries []domain.TaskHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	executor := tx.ExecutorFromContext(ctx, r.pool)

	placeholders := make([]string, 0, len(entries))
	args := make([]any, 0, len(entries)*5)
	for _, entry := range entries {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
		args = append(args, entry.TaskID, entry.ChangedBy, entry.Field, entry.OldValue, entry.NewValue)
	}

	_, err := executor.ExecContext(ctx,
		`INSERT INTO task_history (task_id, changed_by, field, old_value, new_value) VALUES `+
			strings.Join(placeholders, ", "),
		args...,
	)
	if err != nil {
		return fmt.Errorf("inserting task history entries: %w", err)
	}
	return nil
}

func (r *Repository) ListByTask(ctx context.Context, taskID int64) ([]domain.TaskHistoryEntry, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	rows, err := executor.QueryContext(ctx,
		`SELECT id, task_id, changed_by, field, old_value, new_value, changed_at
		 FROM task_history WHERE task_id = ? ORDER BY changed_at, id`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("selecting task history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []domain.TaskHistoryEntry
	for rows.Next() {
		var entry entryEntity
		if err := rows.Scan(&entry.ID, &entry.TaskID, &entry.ChangedBy, &entry.Field,
			&entry.OldValue, &entry.NewValue, &entry.ChangedAt); err != nil {
			return nil, fmt.Errorf("scanning task history row: %w", err)
		}
		entries = append(entries, entry.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task history rows: %w", err)
	}
	return entries, nil
}
