package historyrepo

import (
	"database/sql"
	"time"

	"team-taskflow/internal/domain"
)

type entryEntity struct {
	ID        int64          `db:"id"`
	TaskID    int64          `db:"task_id"`
	ChangedBy int64          `db:"changed_by"`
	Field     string         `db:"field"`
	OldValue  sql.NullString `db:"old_value"`
	NewValue  sql.NullString `db:"new_value"`
	ChangedAt time.Time      `db:"changed_at"`
}

func (e entryEntity) toDomain() domain.TaskHistoryEntry {
	return domain.TaskHistoryEntry{
		ID:        e.ID,
		TaskID:    e.TaskID,
		ChangedBy: e.ChangedBy,
		Field:     e.Field,
		OldValue:  e.OldValue.String,
		NewValue:  e.NewValue.String,
		ChangedAt: e.ChangedAt,
	}
}
