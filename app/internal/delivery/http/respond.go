package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"team-taskflow/internal/domain"
)

func respondJSON(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.ErrorContext(ctx, "encoding response body", "error", err)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return domain.NewValidationError("invalid JSON request body")
	}
	return nil
}
