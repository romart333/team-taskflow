package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/team_create"
)

type teamCreateUsecase interface {
	Handle(ctx context.Context, in team_create.Input) (team_create.Output, error)
}

type teamCreateRequest struct {
	Name string `json:"name"`
}

type teamCreateResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type TeamCreateHandler struct {
	usecase teamCreateUsecase
}

func NewTeamCreateHandler(usecase teamCreateUsecase) *TeamCreateHandler {
	return &TeamCreateHandler{usecase: usecase}
}

func (h *TeamCreateHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	var req teamCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, team_create.Input{ActorID: actor, Name: req.Name})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusCreated, teamCreateResponse{ID: out.TeamID, Name: out.Name})
}
