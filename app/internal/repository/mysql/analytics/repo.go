// Package analyticsrepo implements the reporting reads: team stats, top
// task creators and the assignee integrity audit.
package analyticsrepo

import (
	"context"
	"database/sql"
	"fmt"

	"team-taskflow/internal/domain"
)

type Repository struct {
	pool *sql.DB
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool}
}

// TeamStats reports, per team of the actor, the name, member count and tasks
// done within the window.
func (r *Repository) TeamStats(ctx context.Context, actorID int64, doneWindowDays int) ([]domain.TeamStats, error) {
	rows, err := r.pool.QueryContext(ctx, `
		SELECT t.id,
		       t.name,
		       COUNT(DISTINCT tm.user_id) AS member_count,
		       COUNT(DISTINCT CASE
		           WHEN ta.status = 'done'
		            AND ta.updated_at >= NOW() - INTERVAL ? DAY
		           THEN ta.id END) AS done_tasks
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks ta        ON ta.team_id = t.id
		WHERE EXISTS (
		    SELECT 1 FROM team_members am
		    WHERE am.team_id = t.id AND am.user_id = ?
		)
		GROUP BY t.id, t.name
		ORDER BY t.id`,
		doneWindowDays, actorID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying team stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []domain.TeamStats
	for rows.Next() {
		var s domain.TeamStats
		if err := rows.Scan(&s.TeamID, &s.TeamName, &s.MemberCount, &s.DoneTasksInWindow); err != nil {
			return nil, fmt.Errorf("scanning team stats row: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating team stats rows: %w", err)
	}
	return stats, nil
}

// TopCreators returns the top-N task creators within the window per team the
// actor belongs to. Ties are broken by user ID to keep the ranking deterministic.
func (r *Repository) TopCreators(ctx context.Context, actorID int64, windowDays, limit int) ([]domain.TeamTopCreator, error) {
	rows, err := r.pool.QueryContext(ctx, `
		SELECT team_id, team_name, user_id, user_name, created_count, rnk
		FROM (
		    SELECT t.id   AS team_id,
		           t.name AS team_name,
		           u.id   AS user_id,
		           u.name AS user_name,
		           COUNT(ta.id) AS created_count,
		           ROW_NUMBER() OVER (
		               PARTITION BY t.id
		               ORDER BY COUNT(ta.id) DESC, u.id
		           ) AS rnk
		    FROM teams t
		    JOIN tasks ta ON ta.team_id = t.id
		                 AND ta.created_at >= NOW() - INTERVAL ? DAY
		    JOIN users u  ON u.id = ta.created_by
		    WHERE EXISTS (
		        SELECT 1 FROM team_members am
		        WHERE am.team_id = t.id AND am.user_id = ?
		    )
		    GROUP BY t.id, t.name, u.id, u.name
		) ranked
		WHERE rnk <= ?
		ORDER BY team_id, rnk`,
		windowDays, actorID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying top creators: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var creators []domain.TeamTopCreator
	for rows.Next() {
		var c domain.TeamTopCreator
		if err := rows.Scan(&c.TeamID, &c.TeamName, &c.UserID, &c.UserName, &c.CreatedCount, &c.Rank); err != nil {
			return nil, fmt.Errorf("scanning top creator row: %w", err)
		}
		creators = append(creators, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating top creator rows: %w", err)
	}
	return creators, nil
}

// OrphanedAssignees returns tasks in the actor's teams whose assignee is not
// a member of the task's team, so integrity violations surface before they
// confuse users.
func (r *Repository) OrphanedAssignees(ctx context.Context, actorID int64) ([]domain.OrphanedAssigneeTask, error) {
	rows, err := r.pool.QueryContext(ctx, `
		SELECT ta.id, ta.title, ta.team_id, ta.assignee_id
		FROM tasks ta
		WHERE ta.assignee_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1
		      FROM team_members tm
		      WHERE tm.team_id = ta.team_id
		        AND tm.user_id = ta.assignee_id
		  )
		  AND EXISTS (
		      SELECT 1 FROM team_members am
		      WHERE am.team_id = ta.team_id AND am.user_id = ?
		  )
		ORDER BY ta.id`,
		actorID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying orphaned assignees: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []domain.OrphanedAssigneeTask
	for rows.Next() {
		var t domain.OrphanedAssigneeTask
		if err := rows.Scan(&t.TaskID, &t.Title, &t.TeamID, &t.AssigneeID); err != nil {
			return nil, fmt.Errorf("scanning orphaned assignee row: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating orphaned assignee rows: %w", err)
	}
	return tasks, nil
}
