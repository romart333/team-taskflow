package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/team_list"
)

type teamListUsecase interface {
	Handle(ctx context.Context, in team_list.Input) (team_list.Output, error)
}

type teamListItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type teamListResponse struct {
	Teams []teamListItem `json:"teams"`
}

type TeamListHandler struct {
	usecase teamListUsecase
}

func NewTeamListHandler(usecase teamListUsecase) *TeamListHandler {
	return &TeamListHandler{usecase: usecase}
}

func (h *TeamListHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, team_list.Input{ActorID: actor})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	items := make([]teamListItem, 0, len(out.Teams))
	for _, team := range out.Teams {
		items = append(items, teamListItem{
			ID:   team.Team.ID,
			Name: team.Team.Name,
			Role: string(team.Role),
		})
	}
	respondJSON(ctx, w, http.StatusOK, teamListResponse{Teams: items})
}
