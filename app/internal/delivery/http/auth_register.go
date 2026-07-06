package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/auth_register"
)

type registerUsecase interface {
	Handle(ctx context.Context, in auth_register.Input) (auth_register.Output, error)
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type registerResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type RegisterHandler struct {
	usecase registerUsecase
}

func NewRegisterHandler(usecase registerUsecase) *RegisterHandler {
	return &RegisterHandler{usecase: usecase}
}

func (h *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, auth_register.Input{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusCreated, registerResponse{
		ID:    out.UserID,
		Email: out.Email,
		Name:  out.Name,
	})
}
