package domain

// Reporting windows and limits required by the analytics spec.
const (
	TeamStatsDoneWindowDays = 7
	TopCreatorsWindowDays   = 30
	TopCreatorsLimit        = 3
)

// TeamStats aggregates a team's size and recent throughput. The reporting
// window is parameterized (TeamStatsDoneWindowDays by default).
type TeamStats struct {
	TeamID            int64
	TeamName          string
	MemberCount       int64
	DoneTasksInWindow int64
}

// TeamTopCreator is one entry of the per-team top task creators report.
type TeamTopCreator struct {
	TeamID       int64
	TeamName     string
	UserID       int64
	UserName     string
	CreatedCount int64
	Rank         int
}

// OrphanedAssigneeTask is a task whose assignee is not a member of the
// task's team (integrity audit finding).
type OrphanedAssigneeTask struct {
	TaskID     int64
	Title      string
	TeamID     int64
	AssigneeID int64
}
