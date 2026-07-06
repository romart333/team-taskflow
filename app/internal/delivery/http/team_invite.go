package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/team_invite"
)

type teamInviteUsecase interface {
	Handle(ctx context.Context, in team_invite.Input) (team_invite.Output, error)
}

type teamInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

type teamInviteResponse struct {
	TeamID int64  `json:"team_id"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type TeamInviteHandler struct {
	usecase teamInviteUsecase
}

func NewTeamInviteHandler(usecase teamInviteUsecase) *TeamInviteHandler {
	return &TeamInviteHandler{usecase: usecase}
}

func (h *TeamInviteHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actor, err := actorID(r)
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	teamID, err := pathID(r, "id")
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	var req teamInviteRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, team_invite.Input{
		ActorID: actor,
		TeamID:  teamID,
		Email:   req.Email,
		Role:    req.Role,
	})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusOK, teamInviteResponse{
		TeamID: out.TeamID,
		UserID: out.UserID,
		Role:   string(out.Role),
	})
}
