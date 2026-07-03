package domain

import "time"

// TaskHistoryEntry is one audited change of a task field.
type TaskHistoryEntry struct {
	ID        int64
	TaskID    int64
	ChangedBy int64
	Field     string
	OldValue  string
	NewValue  string
	ChangedAt time.Time
}
