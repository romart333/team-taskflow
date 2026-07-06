package task_update

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"team-taskflow/internal/domain"
)

var fixedNow = time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

// passthroughTx sets up a MockTxManager whose Do runs the callback directly,
// mirroring the real transaction boundary without a database.
func passthroughTx(t *testing.T) *MockTxManager {
	t.Helper()
	tx := NewMockTxManager(t)
	tx.EXPECT().Do(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	return tx
}

func newUsecase(
	tasks TaskRepository, access TeamAccess, history HistoryRepository, tx TxManager, cache TaskCacheInvalidator,
) *Usecase {
	return New(tasks, access, history, tx, cache, func() time.Time { return fixedNow })
}

func TestUsecase_Handle(t *testing.T) {
	baseTask := domain.Task{
		ID: 7, TeamID: 1, Title: "Old title", Description: "Old desc",
		Status: domain.TaskStatusTodo, CreatedBy: 5,
	}

	t.Run("changes are read, written and audited inside one transaction", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil).Once()
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		var gotUpdate domain.Task
		tasks.EXPECT().Update(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
			return task.Title == "New title" && task.Status == domain.TaskStatusDone
		})).Run(func(_ context.Context, task domain.Task) { gotUpdate = task }).Return(nil)

		history.EXPECT().AddEntries(mock.Anything, mock.MatchedBy(func(entries []domain.TaskHistoryEntry) bool {
			return assert.ObjectsAreEqual([]domain.TaskHistoryEntry{
				{TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldTitle, OldValue: "Old title", NewValue: "New title"},
				{TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldStatus, OldValue: "todo", NewValue: "done"},
			}, entries)
		})).Return(nil)

		tasks.EXPECT().GetByID(mock.Anything, int64(7)).RunAndReturn(
			func(context.Context, int64) (domain.Task, error) { return gotUpdate, nil },
		)
		cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil).Once()

		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7,
			Title:  new("New title"),
			Status: new("done"),
		})

		require.NoError(t, err)
		assert.Equal(t, "New title", out.Task.Title)
		assert.Equal(t, domain.TaskStatusDone, out.Task.Status)
		require.NotNil(t, out.Task.CompletedAt, "completion must be stamped on the move into done")
		assert.Equal(t, fixedNow, *out.Task.CompletedAt)
	})

	t.Run("no-op update writes no history and keeps the cache", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("Old title"),
		})

		require.NoError(t, err)
		assert.Equal(t, baseTask, out.Task)
	})

	t.Run("task not found", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(domain.Task{}, domain.ErrNotFound)

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{ActorID: 5, TaskID: 7})

		require.ErrorIs(t, err, domain.ErrNotFound)
		var safeErr *domain.SafeError
		require.ErrorAs(t, err, &safeErr)
		assert.Equal(t, "task not found", safeErr.Msg)
	})

	t.Run("non-member is rejected", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).
			Return(domain.NewPermissionDeniedError("not a member"))

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("x"),
		})

		require.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("invalid status", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Status: new("archived"),
		})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("assignee outside the team", func(t *testing.T) {
		outsider := int64(99)
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		access.EXPECT().EnsureAssigneeMember(mock.Anything, int64(1), outsider).
			Return(domain.NewValidationError("assignee is not a member of this team"))

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &outsider,
		})

		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("explicit null unassigns the task and is audited", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member

		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(assigned, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		var gotUpdate domain.Task
		tasks.EXPECT().Update(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
			return task.AssigneeID == nil
		})).Run(func(_ context.Context, task domain.Task) { gotUpdate = task }).Return(nil)

		history.EXPECT().AddEntries(mock.Anything, []domain.TaskHistoryEntry{
			{TaskID: 7, ChangedBy: 5, Field: domain.TaskFieldAssignee, OldValue: "9", NewValue: ""},
		}).Return(nil)

		tasks.EXPECT().GetByID(mock.Anything, int64(7)).RunAndReturn(
			func(context.Context, int64) (domain.Task, error) { return gotUpdate, nil },
		)
		cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil)

		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: nil,
		})

		require.NoError(t, err)
		assert.Nil(t, out.Task.AssigneeID)
	})

	t.Run("absent assignee field leaves the assignee unchanged", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member

		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(assigned, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		var gotUpdate domain.Task
		tasks.EXPECT().Update(mock.Anything, mock.Anything).
			Run(func(_ context.Context, task domain.Task) { gotUpdate = task }).Return(nil)
		history.EXPECT().AddEntries(mock.Anything, mock.Anything).Return(nil)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).RunAndReturn(
			func(context.Context, int64) (domain.Task, error) { return gotUpdate, nil },
		)
		cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil)

		// No EnsureAssigneeMember expectation: it must not be called.
		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("New title"),
		})

		require.NoError(t, err)
		require.NotNil(t, out.Task.AssigneeID)
		assert.Equal(t, member, *out.Task.AssigneeID)
	})

	t.Run("re-assigning the same member skips the membership lookup", func(t *testing.T) {
		member := int64(9)
		assigned := baseTask
		assigned.AssigneeID = &member

		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(assigned, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		// No EnsureAssigneeMember, Update, AddEntries or InvalidateTeam
		// expectations: re-assigning the same member is a no-op change.
		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &member,
		})

		require.NoError(t, err)
		require.NotNil(t, out.Task.AssigneeID)
		assert.Equal(t, member, *out.Task.AssigneeID)
	})
}

func TestUsecase_Handle_RepositoryFailures(t *testing.T) {
	baseTask := domain.Task{ID: 7, TeamID: 1, Title: "Old", Status: domain.TaskStatusTodo}
	dbErr := errors.New("db down")

	t.Run("membership check failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(dbErr)

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("x"),
		})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("assignee check failure", func(t *testing.T) {
		outsider := int64(99)
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		access.EXPECT().EnsureAssigneeMember(mock.Anything, int64(1), outsider).Return(dbErr)

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, SetAssignee: true, AssigneeID: &outsider,
		})
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("update failure rolls back and skips history and cache", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		tasks.EXPECT().Update(mock.Anything, mock.Anything).Return(dbErr)

		// No AddEntries or InvalidateTeam expectations: a failed write must
		// not be audited or invalidate the cache.
		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("x"),
		})
		require.Error(t, err)
	})

	t.Run("history write failure", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)
		tasks.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
		history.EXPECT().AddEntries(mock.Anything, mock.Anything).Return(dbErr)

		_, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Title: new("x"),
		})
		require.Error(t, err)
	})

	t.Run("description change is audited", func(t *testing.T) {
		tasks := NewMockTaskRepository(t)
		access := NewMockTeamAccess(t)
		history := NewMockHistoryRepository(t)
		cache := NewMockTaskCacheInvalidator(t)
		tx := passthroughTx(t)

		tasks.EXPECT().GetByIDForUpdate(mock.Anything, int64(7)).Return(baseTask, nil)
		access.EXPECT().EnsureTeamMember(mock.Anything, int64(1), int64(5)).Return(nil)

		var gotUpdate domain.Task
		tasks.EXPECT().Update(mock.Anything, mock.Anything).
			Run(func(_ context.Context, task domain.Task) { gotUpdate = task }).Return(nil)
		var gotEntries []domain.TaskHistoryEntry
		history.EXPECT().AddEntries(mock.Anything, mock.MatchedBy(func(entries []domain.TaskHistoryEntry) bool {
			gotEntries = entries
			return true
		})).Return(nil)
		tasks.EXPECT().GetByID(mock.Anything, int64(7)).RunAndReturn(
			func(context.Context, int64) (domain.Task, error) { return gotUpdate, nil },
		)
		cache.EXPECT().InvalidateTeam(mock.Anything, int64(1)).Return(nil)

		out, err := newUsecase(tasks, access, history, tx, cache).Handle(context.Background(), Input{
			ActorID: 5, TaskID: 7, Description: new("new desc"),
		})
		require.NoError(t, err)
		assert.Equal(t, "new desc", out.Task.Description)
		require.Len(t, gotEntries, 1)
		assert.Equal(t, domain.TaskFieldDescription, gotEntries[0].Field)
	})
}
