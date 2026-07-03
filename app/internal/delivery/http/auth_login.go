package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/auth_login"
)

type loginUsecase interface {
	Handle(ctx context.Context, in auth_login.Input) (auth_login.Output, error)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

type LoginHandler struct {
	usecase loginUsecase
}

func NewLoginHandler(usecase loginUsecase) *LoginHandler {
	return &LoginHandler{usecase: usecase}
}

func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, auth_login.Input{Email: req.Email, Password: req.Password})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusOK, loginResponse{AccessToken: out.AccessToken})
}
