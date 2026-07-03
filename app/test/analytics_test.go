//go:build integration

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
	analyticsrepo "team-taskflow/internal/repository/mysql/analytics"
	taskrepo "team-taskflow/internal/repository/mysql/task"
	teamrepo "team-taskflow/internal/repository/mysql/team"
)

func TestAnalyticsQueries(t *testing.T) {
	ctx := context.Background()
	analytics := analyticsrepo.NewRepository(pool)
	tasks := taskrepo.NewRepository(pool)
	teams := teamrepo.NewRepository(pool)

	ownerID := createUser(t, "AnalyticsOwner")
	mateID := createUser(t, "AnalyticsMate")
	teamID := createTeamWithOwner(t, "Analytics", ownerID)
	require.NoError(t, teams.AddMember(ctx, domain.TeamMember{TeamID: teamID, UserID: mateID, Role: domain.RoleMember}))

	// Owner creates two tasks (one done), mate creates one.
	doneID, err := tasks.Create(ctx, domain.Task{
		TeamID: teamID, Title: "done-task", Description: "d",
		Status: domain.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)
	doneTask, err := tasks.GetByID(ctx, doneID)
	require.NoError(t, err)
	doneTask.Status = domain.TaskStatusDone
	require.NoError(t, tasks.Update(ctx, doneTask))

	_, err = tasks.Create(ctx, domain.Task{
		TeamID: teamID, Title: "todo-task", Description: "d",
		Status: domain.TaskStatusTodo, CreatedBy: ownerID,
	})
	require.NoError(t, err)
	_, err = tasks.Create(ctx, domain.Task{
		TeamID: teamID, Title: "mate-task", Description: "d",
		Status: domain.TaskStatusTodo, CreatedBy: mateID,
	})
	require.NoError(t, err)

	t.Run("team stats aggregates members and done tasks", func(t *testing.T) {
		stats, err := analytics.TeamStats(ctx, domain.TeamStatsDoneWindowDays)
		require.NoError(t, err)

		var found *domain.TeamStats
		for i := range stats {
			if stats[i].TeamID == teamID {
				found = &stats[i]
			}
		}
		require.NotNil(t, found, "team must be present in stats")
		assert.EqualValues(t, 2, found.MemberCount)
		assert.EqualValues(t, 1, found.DoneTasksLast7Days)
	})

	t.Run("top creators ranks by created count", func(t *testing.T) {
		creators, err := analytics.TopCreators(ctx, domain.TopCreatorsWindowDays, domain.TopCreatorsLimit)
		require.NoError(t, err)

		var teamCreators []domain.TeamTopCreator
		for _, c := range creators {
			if c.TeamID == teamID {
				teamCreators = append(teamCreators, c)
			}
		}
		require.Len(t, teamCreators, 2)
		assert.Equal(t, ownerID, teamCreators[0].UserID)
		assert.EqualValues(t, 2, teamCreators[0].CreatedCount)
		assert.Equal(t, 1, teamCreators[0].Rank)
		assert.Equal(t, mateID, teamCreators[1].UserID)
		assert.Equal(t, 2, teamCreators[1].Rank)
	})

	t.Run("orphaned assignees finds integrity violations", func(t *testing.T) {
		orphanTaskID, err := tasks.Create(ctx, domain.Task{
			TeamID: teamID, Title: "orphan", Description: "d",
			Status: domain.TaskStatusTodo, AssigneeID: &mateID, CreatedBy: ownerID,
		})
		require.NoError(t, err)

		// Simulate the mate leaving the team, orphaning the assignment.
		_, err = pool.ExecContext(ctx,
			`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, mateID)
		require.NoError(t, err)

		orphans, err := analytics.OrphanedAssignees(ctx)
		require.NoError(t, err)

		var found bool
		for _, o := range orphans {
			if o.TaskID == orphanTaskID {
				found = true
				assert.Equal(t, teamID, o.TeamID)
				assert.Equal(t, mateID, o.AssigneeID)
			}
		}
		assert.True(t, found, "orphaned task must be reported")
	})
}
