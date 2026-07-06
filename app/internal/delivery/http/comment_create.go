package http

import (
	"context"
	"net/http"
	"time"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/usecase/comment_create"
)

type commentCreateUsecase interface {
	Handle(ctx context.Context, in comment_create.Input) (comment_create.Output, error)
}

type commentCreateRequest struct {
	Body string `json:"body"`
}

type commentResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func toCommentResponse(comment domain.TaskComment) commentResponse {
	return commentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		UserID:    comment.UserID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
	}
}

type CommentCreateHandler struct {
	usecase commentCreateUsecase
}

func NewCommentCreateHandler(usecase commentCreateUsecase) *CommentCreateHandler {
	return &CommentCreateHandler{usecase: usecase}
}

func (h *CommentCreateHandler) Handle(w http.ResponseWriter, r *http.Request) {
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

	var req commentCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(ctx, w, err)
		return
	}

	out, err := h.usecase.Handle(ctx, comment_create.Input{ActorID: actor, TaskID: taskID, Body: req.Body})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	respondJSON(ctx, w, http.StatusCreated, toCommentResponse(out.Comment))
}
