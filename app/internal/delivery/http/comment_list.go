package http

import (
	"context"
	"net/http"

	"team-taskflow/internal/usecase/comment_list"
)

type commentListUsecase interface {
	Handle(ctx context.Context, in comment_list.Input) (comment_list.Output, error)
}

type commentListResponse struct {
	Comments []commentResponse `json:"comments"`
}

type CommentListHandler struct {
	usecase commentListUsecase
}

func NewCommentListHandler(usecase commentListUsecase) *CommentListHandler {
	return &CommentListHandler{usecase: usecase}
}

func (h *CommentListHandler) Handle(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.usecase.Handle(ctx, comment_list.Input{ActorID: actor, TaskID: taskID})
	if err != nil {
		respondError(ctx, w, err)
		return
	}

	comments := make([]commentResponse, 0, len(out.Comments))
	for _, comment := range out.Comments {
		comments = append(comments, toCommentResponse(comment))
	}
	respondJSON(ctx, w, http.StatusOK, commentListResponse{Comments: comments})
}
