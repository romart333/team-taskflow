package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/domain"
)

type analyticsUsecase interface {
	TeamStats(ctx context.Context) ([]domain.TeamStats, error)
	TopCreators(ctx context.Context) ([]domain.TeamTopCreator, error)
	OrphanedAssignees(ctx context.Context) ([]domain.OrphanedAssigneeTask, error)
}

type teamStatsItem struct {
	TeamID            int64  `json:"team_id"`
	TeamName          string `json:"team_name"`
	MemberCount       int64  `json:"member_count"`
	DoneTasksInWindow int64  `json:"done_tasks_in_window"`
}

type teamStatsResponse struct {
	Teams []teamStatsItem `json:"teams"`
}

type topCreatorItem struct {
	TeamID       int64  `json:"team_id"`
	TeamName     string `json:"team_name"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	CreatedCount int64  `json:"created_count"`
	Rank         int    `json:"rank"`
}

type topCreatorsResponse struct {
	Creators []topCreatorItem `json:"creators"`
}

type orphanedAssigneeItem struct {
	TaskID     int64  `json:"task_id"`
	Title      string `json:"title"`
	TeamID     int64  `json:"team_id"`
	AssigneeID int64  `json:"assignee_id"`
}

type orphanedAssigneesResponse struct {
	Tasks []orphanedAssigneeItem `json:"tasks"`
}

type AnalyticsHandler struct {
	usecase analyticsUsecase
}

func NewAnalyticsHandler(usecase analyticsUsecase) *AnalyticsHandler {
	return &AnalyticsHandler{usecase: usecase}
}

func (h *AnalyticsHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.usecase.TeamStats(ctx)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	items := make([]teamStatsItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, teamStatsItem{
			TeamID:            s.TeamID,
			TeamName:          s.TeamName,
			MemberCount:       s.MemberCount,
			DoneTasksInWindow: s.DoneTasksInWindow,
		})
	}
	respondJSON(ctx, w, http.StatusOK, teamStatsResponse{Teams: items})
}

func (h *AnalyticsHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	creators, err := h.usecase.TopCreators(ctx)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	items := make([]topCreatorItem, 0, len(creators))
	for _, c := range creators {
		items = append(items, topCreatorItem{
			TeamID:       c.TeamID,
			TeamName:     c.TeamName,
			UserID:       c.UserID,
			UserName:     c.UserName,
			CreatedCount: c.CreatedCount,
			Rank:         c.Rank,
		})
	}
	respondJSON(ctx, w, http.StatusOK, topCreatorsResponse{Creators: items})
}

func (h *AnalyticsHandler) OrphanedAssignees(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tasks, err := h.usecase.OrphanedAssignees(ctx)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	items := make([]orphanedAssigneeItem, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, orphanedAssigneeItem{
			TaskID:     t.TaskID,
			Title:      t.Title,
			TeamID:     t.TeamID,
			AssigneeID: t.AssigneeID,
		})
	}
	respondJSON(ctx, w, http.StatusOK, orphanedAssigneesResponse{Tasks: items})
}
