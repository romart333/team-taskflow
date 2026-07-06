package domain

// TaskFilter selects tasks of one team with optional narrowing and pagination.
type TaskFilter struct {
	TeamID     int64
	Status     *TaskStatus
	AssigneeID *int64
	Page       int
	PageSize   int
}

// Offset returns the DB offset for the requested page (pages are 1-based).
func (f TaskFilter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// Normalize clamps pagination to sane bounds.
func (f TaskFilter) Normalize(defaultPageSize, maxPageSize int) TaskFilter {
	normalized := f
	if normalized.Page <= 0 {
		normalized.Page = 1
	}
	if normalized.PageSize <= 0 {
		normalized.PageSize = defaultPageSize
	}
	if normalized.PageSize > maxPageSize {
		normalized.PageSize = maxPageSize
	}
	return normalized
}

// TaskPage is one page of a filtered task listing.
type TaskPage struct {
	Tasks []Task
	Total int64
}
