package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"team-taskflow/internal/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

const internalErrorMessage = "internal server error"

// respondError maps domain errors to HTTP statuses in one place and logs
// every error response with its full context.
func respondError(ctx context.Context, w http.ResponseWriter, err error) {
	status := statusFromError(err)

	if status >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, "request failed", "status", status, "error", err)
	} else {
		slog.WarnContext(ctx, "request rejected", "status", status, "error", err)
	}

	respondJSON(ctx, w, status, errorResponse{Error: clientMessage(err, status)})
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrPermissionDenied):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists), errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// clientMessage never leaks internal error text: it surfaces only messages
// explicitly marked safe (domain.SafeError) or generic sentinel wording.
func clientMessage(err error, status int) string {
	if status >= http.StatusInternalServerError {
		return internalErrorMessage
	}

	if safe, ok := errors.AsType[*domain.SafeError](err); ok {
		return safe.Msg
	}

	for _, sentinel := range []error{
		domain.ErrValidation, domain.ErrUnauthorized, domain.ErrPermissionDenied,
		domain.ErrNotFound, domain.ErrAlreadyExists, domain.ErrConflict, domain.ErrRateLimited,
	} {
		if errors.Is(err, sentinel) {
			return sentinel.Error()
		}
	}
	return internalErrorMessage
}
