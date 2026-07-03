package http

import (
	"context"
	"net/http"
	"strconv"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/usecase/task_list"
)

type taskListUsecase interface {
	Handle(ctx context.Context, in task_list.Input) (task_list.Output, error)
}

type taskListResponse struct {
	Tasks    []taskResponse `json:"tasks"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type TaskListHandler struct {
	usecase taskListUsecase
}

func NewTaskListHandler(usecase taskListUsecase) *TaskListHandler {
	return &TaskListHandler{usecase: usecase}
}

func (h *TaskListHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	filter, err := parseTaskFilter(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, task_list.Input{ActorID: actor, Filter: filter})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	tasks := make([]taskResponse, 0, len(out.Page.Tasks))
	for _, task := range out.Page.Tasks {
		tasks = append(tasks, toTaskResponse(task))
	}
	respondJSON(ctx, w, http.StatusOK, taskListResponse{
		Tasks:    tasks,
		Total:    out.Page.Total,
		Page:     out.PageNum,
		PageSize: out.PageSize,
	})
}

func parseTaskFilter(r *http.Request) (domain.TaskFilter, error) {
	query := r.URL.Query()

	teamID, err := strconv.ParseInt(query.Get("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		return domain.TaskFilter{}, domain.NewValidationError("team_id query parameter is required and must be a positive integer")
	}
	filter := domain.TaskFilter{TeamID: teamID}

	if raw := query.Get("status"); raw != "" {
		status, err := domain.ParseTaskStatus(raw)
		if err != nil {
			return domain.TaskFilter{}, err
		}
		filter.Status = &status
	}

	if raw := query.Get("assignee_id"); raw != "" {
		assigneeID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || assigneeID <= 0 {
			return domain.TaskFilter{}, domain.NewValidationError("assignee_id must be a positive integer")
		}
		filter.AssigneeID = &assigneeID
	}

	if raw := query.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page <= 0 {
			return domain.TaskFilter{}, domain.NewValidationError("page must be a positive integer")
		}
		filter.Page = page
	}

	if raw := query.Get("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil || pageSize <= 0 {
			return domain.TaskFilter{}, domain.NewValidationError("page_size must be a positive integer")
		}
		filter.PageSize = pageSize
	}

	return filter, nil
}
