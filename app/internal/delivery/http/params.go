package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"team-taskflow/internal/domain"
)

// pathID parses a positive integer URL parameter.
func pathID(r *http.Request, name string) (int64, error) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.NewValidationError("invalid " + name + " path parameter")
	}
	return id, nil
}
