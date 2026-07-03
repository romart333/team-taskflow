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
	teams   TeamRepository
	history HistoryRepository
	tx      TxManager
	cache   TaskCacheInvalidator
}

func New(
	tasks TaskRepository,
	teams TeamRepository,
	history HistoryRepository,
	tx TxManager,
	cache TaskCacheInvalidator,
) *Usecase {
	return &Usecase{tasks: tasks, teams: teams, history: history, tx: tx, cache: cache}
}

// Handle updates a task on behalf of a team member and records every field
// change in the task history within the same transaction.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	current, err := u.tasks.GetByID(ctx, in.TaskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("loading task: %w", domain.NewNotFoundError("task not found"))
		}
		return Output{}, fmt.Errorf("loading task: %w", err)
	}

	if _, err := u.teams.GetMember(ctx, current.TeamID, in.ActorID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Output{}, fmt.Errorf("checking membership: %w",
				domain.NewPermissionDeniedError("you are not a member of this task's team"))
		}
		return Output{}, fmt.Errorf("getting membership: %w", err)
	}

	updated, err := u.applyChanges(ctx, current, in)
	if err != nil {
		return Output{}, err
	}

	changes := current.Diff(updated)
	if len(changes) == 0 {
		return Output{Task: current}, nil
	}

	err = u.tx.Do(ctx, func(txCtx context.Context) error {
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
		return nil
	})
	if err != nil {
		return Output{}, fmt.Errorf("updating task transactionally: %w", err)
	}

	if err := u.cache.InvalidateTeam(ctx, current.TeamID); err != nil {
		slog.WarnContext(ctx, "task list cache invalidation failed", "team_id", current.TeamID, "error", err)
	}

	task, err := u.tasks.GetByID(ctx, in.TaskID)
	if err != nil {
		return Output{}, fmt.Errorf("loading updated task: %w", err)
	}

	slog.InfoContext(ctx, "task updated",
		"task_id", in.TaskID, "changed_by", in.ActorID, "changes", len(changes))
	return Output{Task: task}, nil
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
	if in.AssigneeID != nil {
		if _, err := u.teams.GetMember(ctx, current.TeamID, *in.AssigneeID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.Task{}, fmt.Errorf("checking assignee membership: %w",
					domain.NewValidationError("assignee is not a member of this team"))
			}
			return domain.Task{}, fmt.Errorf("getting assignee membership: %w", err)
		}
		updated.AssigneeID = in.AssigneeID
	}

	return updated, nil
}
