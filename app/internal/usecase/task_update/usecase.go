package task_update

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"team-taskflow/internal/domain"
)

type Usecase struct {
	tasks   TaskRepository
	access  TeamAccess
	teams   TeamRepository
	history HistoryRepository
	tx      TxManager
	cache   TaskCacheInvalidator
}

func New(
	tasks TaskRepository,
	access TeamAccess,
	teams TeamRepository,
	history HistoryRepository,
	tx TxManager,
	cache TaskCacheInvalidator,
) *Usecase {
	return &Usecase{tasks: tasks, access: access, teams: teams, history: history, tx: tx, cache: cache}
}

// Handle updates a task on behalf of a team member and records every field
// change in the task history. The snapshot is read with a row lock inside the
// same transaction as the write: concurrent updates of one task serialize
// instead of overwriting each other's fields from stale snapshots.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	var out Output
	var teamID int64
	changeCount := 0

	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		current, err := u.tasks.GetByIDForUpdate(txCtx, in.TaskID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("loading task: %w", domain.NewNotFoundError("task not found"))
			}
			return fmt.Errorf("loading task: %w", err)
		}
		teamID = current.TeamID

		if err := u.access.EnsureTeamMember(txCtx, current.TeamID, in.ActorID); err != nil {
			return fmt.Errorf("authorizing actor: %w", err)
		}

		updated, err := u.applyChanges(txCtx, current, in)
		if err != nil {
			return err
		}

		changes := current.Diff(updated)
		if len(changes) == 0 {
			out = Output{Task: current}
			return nil
		}

		if err := u.tasks.Update(txCtx, updated); err != nil {
			return fmt.Errorf("updating task: %w", err)
		}

		entries := make([]domain.TaskHistoryEntry, 0, len(changes))
		for _, change := range changes {
			entries = append(entries, domain.TaskHistoryEntry{
				TaskID:    in.TaskID,
				ChangedBy: in.ActorID,
				Field:     change.Field,
				OldValue:  change.OldValue,
				NewValue:  change.NewValue,
			})
		}
		if err := u.history.AddEntries(txCtx, entries); err != nil {
			return fmt.Errorf("recording task history: %w", err)
		}

		// Re-read inside the transaction so the response carries the
		// DB-maintained updated_at.
		task, err := u.tasks.GetByID(txCtx, in.TaskID)
		if err != nil {
			return fmt.Errorf("loading updated task: %w", err)
		}
		out = Output{Task: task}
		changeCount = len(changes)
		return nil
	})
	if err != nil {
		return Output{}, fmt.Errorf("updating task transactionally: %w", err)
	}

	if changeCount == 0 {
		return out, nil
	}

	if err := u.cache.InvalidateTeam(ctx, teamID); err != nil {
		slog.WarnContext(ctx, "task list cache invalidation failed", "team_id", teamID, "error", err)
	}

	slog.InfoContext(ctx, "task updated",
		"task_id", in.TaskID, "changed_by", in.ActorID, "changes", changeCount)
	return out, nil
}

// applyChanges builds the updated task from the patch input, validating every
// provided field.
func (u *Usecase) applyChanges(ctx context.Context, current domain.Task, in Input) (domain.Task, error) {
	updated := current

	if in.Title != nil {
		if err := domain.ValidateNewTask(*in.Title); err != nil {
			return domain.Task{}, fmt.Errorf("validating title: %w", err)
		}
		updated.Title = *in.Title
	}
	if in.Description != nil {
		updated.Description = *in.Description
	}
	if in.Status != nil {
		status, err := domain.ParseTaskStatus(*in.Status)
		if err != nil {
			return domain.Task{}, fmt.Errorf("validating status: %w", err)
		}
		updated.Status = status
	}
	if in.SetAssignee {
		if in.AssigneeID != nil && !sameAssignee(current.AssigneeID, in.AssigneeID) {
			if _, err := u.teams.GetMember(ctx, current.TeamID, *in.AssigneeID); err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return domain.Task{}, fmt.Errorf("checking assignee membership: %w",
						domain.NewValidationError("assignee is not a member of this team"))
				}
				return domain.Task{}, fmt.Errorf("getting assignee membership: %w", err)
			}
		}
		updated.AssigneeID = in.AssigneeID
	}

	return updated, nil
}

func sameAssignee(a, b *int64) bool {
	return a != nil && b != nil && *a == *b
}
