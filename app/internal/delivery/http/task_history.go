package http

import (
	"context"
	"net/http"
	"time"

	"team-taskflow/internal/usecase/task_history_get"
)

type taskHistoryUsecase interface {
	Handle(ctx context.Context, in task_history_get.Input) (task_history_get.Output, error)
}

type taskHistoryItem struct {
	ID        int64     `json:"id"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy int64     `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
}

type taskHistoryResponse struct {
	Items []taskHistoryItem `json:"items"`
}

type TaskHistoryHandler struct {
	usecase taskHistoryUsecase
}

func NewTaskHistoryHandler(usecase taskHistoryUsecase) *TaskHistoryHandler {
	return &TaskHistoryHandler{usecase: usecase}
}

func (h *TaskHistoryHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	taskID, err := pathID(r, "id")
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, task_history_get.Input{ActorID: actor, TaskID: taskID})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	items := make([]taskHistoryItem, 0, len(out.Entries))
	for _, entry := range out.Entries {
		items = append(items, taskHistoryItem{
			ID:        entry.ID,
			Field:     entry.Field,
			OldValue:  entry.OldValue,
			NewValue:  entry.NewValue,
			ChangedBy: entry.ChangedBy,
			ChangedAt: entry.ChangedAt,
		})
	}
	respondJSON(ctx, w, http.StatusOK, taskHistoryResponse{Items: items})
}
