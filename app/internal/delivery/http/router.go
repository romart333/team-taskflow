package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RouterDeps carries handlers and middleware required to assemble the router.
type RouterDeps struct {
	AuthMiddleware func(http.Handler) http.Handler

	Register    http.HandlerFunc
	Login       http.HandlerFunc
	TeamCreate  http.HandlerFunc
	TeamList    http.HandlerFunc
	TeamInvite  http.HandlerFunc
	TaskCreate  http.HandlerFunc
	TaskList    http.HandlerFunc
	TaskUpdate  http.HandlerFunc
	TaskHistory http.HandlerFunc

	CommentCreate http.HandlerFunc
	CommentList   http.HandlerFunc

	AnalyticsTeamStats         http.HandlerFunc
	AnalyticsTopCreators       http.HandlerFunc
	AnalyticsOrphanedAssignees http.HandlerFunc
}

// NewRouter assembles the HTTP routing tree.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", handleHealth)

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/register", deps.Register)
		api.Post("/login", deps.Login)

		api.Group(func(protected chi.Router) {
			protected.Use(deps.AuthMiddleware)

			protected.Post("/teams", deps.TeamCreate)
			protected.Get("/teams", deps.TeamList)
			protected.Post("/teams/{id}/invite", deps.TeamInvite)

			protected.Post("/tasks", deps.TaskCreate)
			protected.Get("/tasks", deps.TaskList)
			protected.Put("/tasks/{id}", deps.TaskUpdate)
			protected.Get("/tasks/{id}/history", deps.TaskHistory)

			protected.Post("/tasks/{id}/comments", deps.CommentCreate)
			protected.Get("/tasks/{id}/comments", deps.CommentList)

			protected.Get("/analytics/team-stats", deps.AnalyticsTeamStats)
			protected.Get("/analytics/top-creators", deps.AnalyticsTopCreators)
			protected.Get("/analytics/integrity/orphaned-assignees", deps.AnalyticsOrphanedAssignees)
		})
	})

	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
