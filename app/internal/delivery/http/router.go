package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RouterDeps carries handlers and middleware required to assemble the router.
type RouterDeps struct{}

// NewRouter assembles the HTTP routing tree.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", handleHealth)

	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
