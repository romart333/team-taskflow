package http

import (
	"context"
	"encoding/json"
	"net/http"

	"team-taskflow/internal/usecase/task_update"
)

type taskUpdateUsecase interface {
	Handle(ctx context.Context, in task_update.Input) (task_update.Output, error)
}

// optionalInt64 distinguishes an absent JSON field from an explicit null, so
// PUT bodies can unassign a task with {"assignee_id": null}.
type optionalInt64 struct {
	Set   bool
	Value *int64
}

func (o *optionalInt64) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}

type taskUpdateRequest struct {
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *string       `json:"status,omitempty"`
	AssigneeID  optionalInt64 `json:"assignee_id"`
}

type TaskUpdateHandler struct {
	usecase taskUpdateUsecase
}

func NewTaskUpdateHandler(usecase taskUpdateUsecase) *TaskUpdateHandler {
	return &TaskUpdateHandler{usecase: usecase}
}

func (h *TaskUpdateHandler) Handle(w http.ResponseWriter, r *http.Request) {
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

	var req taskUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, task_update.Input{
		ActorID:     actor,
		TaskID:      taskID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		SetAssignee: req.AssigneeID.Set,
		AssigneeID:  req.AssigneeID.Value,
	})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusOK, toTaskResponse(out.Task))
}
