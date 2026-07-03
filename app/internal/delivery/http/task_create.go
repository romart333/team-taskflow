package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/task_create"
)

type taskCreateUsecase interface {
	Handle(ctx context.Context, in task_create.Input) (task_create.Output, error)
}

type taskCreateRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  *int64 `json:"assignee_id,omitempty"`
}

type TaskCreateHandler struct {
	usecase taskCreateUsecase
}

func NewTaskCreateHandler(usecase taskCreateUsecase) *TaskCreateHandler {
	return &TaskCreateHandler{usecase: usecase}
}

func (h *TaskCreateHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	var req taskCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, task_create.Input{
		ActorID:     actor,
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusCreated, toTaskResponse(out.Task))
}
