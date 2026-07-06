package task_history_get

import (
	"context"
	"fmt"
)

type Usecase struct {
	access  TaskAccess
	history HistoryRepository
}

func New(access TaskAccess, history HistoryRepository) *Usecase {
	return &Usecase{access: access, history: history}
}

// Handle returns the change history of a task visible to team members only.
func (u *Usecase) Handle(ctx context.Context, in Input) (Output, error) {
	if _, err := u.access.LoadTaskForMember(ctx, in.TaskID, in.ActorID); err != nil {
		return Output{}, fmt.Errorf("authorizing task access: %w", err)
	}

	entries, err := u.history.ListByTask(ctx, in.TaskID)
	if err != nil {
		return Output{}, fmt.Errorf("listing task history: %w", err)
	}
	return Output{Entries: entries}, nil
}
