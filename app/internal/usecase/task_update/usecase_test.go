package task_update

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

var fixedNow = time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

type taskRepoMock struct {
	task      domain.Task
	getErr    error
	updateErr error
	gotUpdate *domain.Task
	lockCalls int
}

func (m *taskRepoMock) GetByIDForUpdate(context.Context, int64) (domain.Task, error) {
	m.lockCalls++
	return m.get()
}

func (m *taskRepoMock) GetByID(context.Context, int64) (domain.Task, error) {
	return m.get()
}

func (m *taskRepoMock) get() (domain.Task, error) {
	if m.gotUpdate != nil {
		return *m.gotUpdate, m.getErr
	}
	return m.task, m.getErr
}

func (m *taskRepoMock) Update(_ context.Context, task domain.Task) error {
	if m.updateErr == nil {
		m.gotUpdate = &task
	}
	return m.updateErr
}

type accessMock struct {
	err   error
	calls int
}

func (m *accessMock) EnsureTeamMember(context.Context, int64, int64) error {
	m.calls++
	return m.err
}

type teamRepoMock struct {
	errs  map[int64]error
	calls int
}

func (m *teamRepoMock) GetMember(_ context.Context, _ int64, userID int64) (domain.TeamMember, error) {
	m.calls++
	if err, ok := m.errs[userID]; ok {
		return domain.TeamMember{}, err
	}
	return domain.TeamMember{UserID: userID, Role: domain.RoleMember}, nil
}

type historyRepoMock struct {
	entries []domain.TaskHistoryEntry
	err     error
}

func (m *historyRepoMock) AddEntries(_ context.Context, entries []domain.TaskHistoryEntry) error {
	m.entries = append(m.entries, entries...)
	return m.err
}

type txMock struct{ calls int }

func (m *txMock) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type invalidatorMock struct{ calls int }

func (m *invalidatorMock) InvalidateTeam(context.Context, int64) error {
	m.calls++
	return nil
}

type fixture struct {
	tasks   *taskRepoMock
	access  *accessMock
	teams   *teamRepoMock
	history *historyRepoMock
	tx      *txMock
	cache   *invalidatorMock
}

func newFixture(task domain.Task) *fixture {
	return &fixture{
		tasks:   &taskRepoMock{task: task},
		access:  &accessMock{},
		teams:   &teamRepoMock{},
		history: &historyRepoMock{},
		tx:      &txMock{},
		cache:   &invalidatorMock{},
	}
}

func (f *fixture) usecase() *Usecase {
	return New(f.tasks, f.access, f.teams, f.history, f.tx, f.cache, func() time.Time { return fixedNow })
}

func TestUsecase_Handle(t *testing.T) {
	baseTask := domain.Task{
		ID: 7, TeamID: 1, Title: "Old title", Description: "Old desc",
		Status: domain.TaskStatusTodo, CreatedBy: 5,
	}

	t.Run("changes are read, written and audited inside one transaction", func(t *testing.T) {
		f := newFixture(baseTask)

		out, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7,
			Title:  new("New title"),
			Status: new("done"),
		})

		require.NoError(t, err)
		assert.Equal(t, "New title", out.Task.Title)
		assert.Equal(t, domain.TaskStatusDone, out.Task.Status)
		require.NotNil(t, out.Task.CompletedAt, "completion must be stamped on the move into done")
		assert.Equal(t, fixedNow, *out.Task.CompletedAt)
		assert.Equal(t, 1, f.tx.calls)
		assert.Equal(t, 1, f.tasks.lockCalls, "snapshot must be read with a row lock")
		assert.Equal(t, 1, f.cache.calls)
		require.Len(t, f.history.entries, 2)
		assert.Equal(t, domain.TaskHistoryEntry{
			TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldTitle,
			OldValue: "Old title", NewValue: "New title",
		}, f.history.entries[0])
		assert.Equal(t, domain.TaskHistoryEntry{
			TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldStatus,
			OldValue: "todo", NewValue: "done",
		}, f.history.entries[1])
	})

	t.Run("no-op update writes no history and keeps the cache", func(t *testing.T) {
		f := newFixture(baseTask)

		out, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("Old title"),
		})

		require.NoError(t, err)
		assert.Equal(t, baseTask, out.Task)
		assert.Nil(t, f.tasks.gotUpdate, "no write must happen")
		assert.Empty(t, f.history.entries)
		assert.Zero(t, f.cache.calls)
	})

	t.Run("task not found", func(t *testing.T) {
		f := newFixture(baseTask)
		f.tasks = &taskRepoMock{getErr: domain.ErrNotFound}

		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
		var safeErr *domain.SafeError
		require.ErrorAs(t, err, &safeErr)
		assert.Equal(t, "task not found", safeErr.Msg)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		f := newFixture(baseTask)
		f.access = &accessMock{err: domain.NewPermissionDeniedError("not a member")}

		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Title: new("x")})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("invalid status", func(t *testing.T) {
		f := newFixture(baseTask)

		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Status: new("archived")})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("assignee outside the team", func(t *testing.T) {
		outsider := int64(99)
		f := newFixture(baseTask)
		f.teams = &teamRepoMock{errs: map[int64]error{99: domain.ErrNotFound}}

		_, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &outsider,
		})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("explicit null unassigns the task and is audited", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member
		f := newFixture(assigned)

		out, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: nil,
		})

		require.NoError(t, err)
		assert.Nil(t, out.Task.AssigneeID)
		require.Len(t, f.history.entries, 1)
		assert.Equal(t, domain.TaskHistoryEntry{
			TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldAssignee,
			OldValue: "9", NewValue: "",
		}, f.history.entries[0])
	})

	t.Run("absent assignee field leaves the assignee unchanged", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member
		f := newFixture(assigned)

		out, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("New title"),
		})

		require.NoError(t, err)
		require.NotNil(t, out.Task.AssigneeID)
		assert.Equal(t, member, *out.Task.AssigneeID)
		assert.Zero(t, f.teams.calls, "no assignee membership lookup expected")
	})

	t.Run("re-assigning the same member skips the membership lookup", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member
		f := newFixture(assigned)

		out, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &member,
		})

		require.NoError(t, err)
		require.NotNil(t, out.Task.AssigneeID)
		assert.Equal(t, member, *out.Task.AssigneeID)
		assert.Zero(t, f.teams.calls, "unchanged assignee must not be re-validated")
		assert.Empty(t, f.history.entries)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	baseTask := domain.Task{ID: 7, TeamID: 1, Title: "Old", Status: domain.TaskStatusTodo}
	dbErr := errors.New("db down")

	t.Run("membership check failure", func(t *testing.T) {
		f := newFixture(baseTask)
		f.access = &accessMock{err: dbErr}
		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Title: new("x")})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("assignee check failure", func(t *testing.T) {
		outsider := int64(99)
		f := newFixture(baseTask)
		f.teams = &teamRepoMock{errs: map[int64]error{99: dbErr}}
		_, err := f.usecase().Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &outsider,
		})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("update failure rolls back and skips history and cache", func(t *testing.T) {
		f := newFixture(baseTask)
		f.tasks = &taskRepoMock{task: baseTask, updateErr: dbErr}
		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Title: new("x")})
		require.Error(t, err)
		assert.Empty(t, f.history.entries)
		assert.Zero(t, f.cache.calls)
	})

	t.Run("history write failure", func(t *testing.T) {
		f := newFixture(baseTask)
		f.history = &historyRepoMock{err: dbErr}
		_, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Title: new("x")})
		require.Error(t, err)
	})

	t.Run("description change is audited", func(t *testing.T) {
		f := newFixture(baseTask)
		out, err := f.usecase().Handle(context.Background(), Input{ActorID: 5, TaskID: 7, Description: new("new desc")})
		require.NoError(t, err)
		assert.Equal(t, "new desc", out.Task.Description)
		require.Len(t, f.history.entries, 1)
		assert.Equal(t, domain.TaskFieldDescription, f.history.entries[0].Field)
	})
}
