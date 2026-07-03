//go:build integration

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/infrastructure/tx"
	commentrepo "team-taskflow/internal/repository/mysql/comment"
	historyrepo "team-taskflow/internal/repository/mysql/history"
	taskrepo "team-taskflow/internal/repository/mysql/task"
	teamrepo "team-taskflow/internal/repository/mysql/team"
	userrepo "team-taskflow/internal/repository/mysql/user"
)

var userSeq int

// createUser inserts a user with a unique email and returns its ID.
func createUser(t *testing.T, name string) int64 {
	t.Helper()
	userSeq++
	id, err := userrepo.NewRepository(pool).Create(context.Background(), domain.User{
		Email:        fmt.Sprintf("user%d@example.com", userSeq),
		Name:         name,
		PasswordHash: "hash",
	})
	require.NoError(t, err)
	return id
}

func createTeamWithOwner(t *testing.T, name string, ownerID int64) int64 {
	t.Helper()
	ctx := context.Background()
	teams := teamrepo.NewRepository(pool)

	teamID, err := teams.CreateTeam(ctx, domain.Team{Name: name, CreatedBy: ownerID})
	require.NoError(t, err)
	require.NoError(t, teams.AddMember(ctx, domain.TeamMember{
		TeamID: teamID, UserID: ownerID, Role: domain.RoleOwner,
	}))
	return teamID
}

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	users := userrepo.NewRepository(pool)

	id := createUser(t, "Alice")

	byID, err := users.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Alice", byID.Name)

	byEmail, err := users.GetByEmail(ctx, byID.Email)
	require.NoError(t, err)
	assert.Equal(t, id, byEmail.ID)

	_, err = users.Create(ctx, domain.User{Email: byID.Email, Name: "Dup", PasswordHash: "h"})
	require.ErrorIs(t, err, domain.ErrAlreadyExists)

	_, err = users.GetByEmail(ctx, "ghost@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestTeamRepository(t *testing.T) {
	ctx := context.Background()
	teams := teamrepo.NewRepository(pool)

	ownerID := createUser(t, "Owner")
	memberID := createUser(t, "Member")
	teamID := createTeamWithOwner(t, "Platform", ownerID)

	require.NoError(t, teams.AddMember(ctx, domain.TeamMember{
		TeamID: teamID, UserID: memberID, Role: domain.RoleMember,
	}))

	err := teams.AddMember(ctx, domain.TeamMember{TeamID: teamID, UserID: memberID, Role: domain.RoleMember})
	require.ErrorIs(t, err, domain.ErrAlreadyExists)

	member, err := teams.GetMember(ctx, teamID, memberID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleMember, member.Role)

	_, err = teams.GetMember(ctx, teamID, 999999)
	require.ErrorIs(t, err, domain.ErrNotFound)

	listed, err := teams.ListByUser(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "Platform", listed[0].Team.Name)
	assert.Equal(t, domain.RoleOwner, listed[0].Role)
}

func TestTaskRepositoryListFilters(t *testing.T) {
	ctx := context.Background()
	tasks := taskrepo.NewRepository(pool)
	teams := teamrepo.NewRepository(pool)

	authorID := createUser(t, "Author")
	assigneeID := createUser(t, "Assignee")
	teamID := createTeamWithOwner(t, "Filters", authorID)
	require.NoError(t, teams.AddMember(ctx, domain.TeamMember{TeamID: teamID, UserID: assigneeID, Role: domain.RoleMember}))

	mkTask := func(title string, status domain.TaskStatus, assignee *int64) int64 {
		id, err := tasks.Create(ctx, domain.Task{
			TeamID: teamID, Title: title, Description: "d",
			Status: domain.TaskStatusTodo, AssigneeID: assignee, CreatedBy: authorID,
		})
		require.NoError(t, err)
		if status != domain.TaskStatusTodo {
			task, err := tasks.GetByID(ctx, id)
			require.NoError(t, err)
			task.Status = status
			require.NoError(t, tasks.Update(ctx, task))
		}
		return id
	}

	mkTask("t1", domain.TaskStatusTodo, nil)
	mkTask("t2", domain.TaskStatusDone, &assigneeID)
	mkTask("t3", domain.TaskStatusDone, &assigneeID)
	mkTask("t4", domain.TaskStatusInProgress, nil)

	all, err := tasks.List(ctx, domain.TaskFilter{TeamID: teamID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 4, all.Total)
	require.Len(t, all.Tasks, 4)
	assert.Equal(t, "t4", all.Tasks[0].Title, "newest first")

	done := domain.TaskStatusDone
	filtered, err := tasks.List(ctx, domain.TaskFilter{
		TeamID: teamID, Status: &done, AssigneeID: &assigneeID, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, filtered.Total)

	paged, err := tasks.List(ctx, domain.TaskFilter{TeamID: teamID, Page: 2, PageSize: 3})
	require.NoError(t, err)
	assert.EqualValues(t, 4, paged.Total)
	assert.Len(t, paged.Tasks, 1, "second page holds the remainder")
}

func TestHistoryAndCommentsRepositories(t *testing.T) {
	ctx := context.Background()
	tasks := taskrepo.NewRepository(pool)
	history := historyrepo.NewRepository(pool)
	comments := commentrepo.NewRepository(pool)

	authorID := createUser(t, "Historian")
	teamID := createTeamWithOwner(t, "History", authorID)
	taskID, err := tasks.Create(ctx, domain.Task{
		TeamID: teamID, Title: "t", Description: "d",
		Status: domain.TaskStatusTodo, CreatedBy: authorID,
	})
	require.NoError(t, err)

	require.NoError(t, history.AddEntries(ctx, []domain.TaskHistoryEntry{
		{TaskID: taskID, ChangedBy: authorID, Field: domain.TaskFieldStatus, OldValue: "todo", NewValue: "done"},
		{TaskID: taskID, ChangedBy: authorID, Field: domain.TaskFieldTitle, OldValue: "t", NewValue: "t2"},
	}))

	entries, err := history.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, domain.TaskFieldStatus, entries[0].Field)

	commentID, err := comments.Create(ctx, domain.TaskComment{TaskID: taskID, UserID: authorID, Body: "hello"})
	require.NoError(t, err)
	comment, err := comments.GetByID(ctx, commentID)
	require.NoError(t, err)
	assert.Equal(t, "hello", comment.Body)

	listed, err := comments.ListByTask(ctx, taskID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
}

func TestTxManagerRollback(t *testing.T) {
	ctx := context.Background()
	manager := tx.NewManager(pool)
	users := userrepo.NewRepository(pool)

	sentinel := fmt.Errorf("abort")
	email := "rollback@example.com"

	err := manager.Do(ctx, func(txCtx context.Context) error {
		if _, err := users.Create(txCtx, domain.User{Email: email, Name: "R", PasswordHash: "h"}); err != nil {
			return err
		}
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	_, err = users.GetByEmail(ctx, email)
	require.ErrorIs(t, err, domain.ErrNotFound, "insert must be rolled back")

	err = manager.Do(ctx, func(txCtx context.Context) error {
		_, err := users.Create(txCtx, domain.User{Email: email, Name: "R", PasswordHash: "h"})
		return err
	})
	require.NoError(t, err)

	_, err = users.GetByEmail(ctx, email)
	require.NoError(t, err, "committed insert must be visible")
}
