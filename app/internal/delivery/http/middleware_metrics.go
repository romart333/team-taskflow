package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// httpMetrics records served requests.
type httpMetrics interface {
	Observe(method, path string, status int, elapsed time.Duration)
}

// statusRecorder captures the response status code for metrics.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// NewMetricsMiddleware records request count and latency per route pattern
// (not per raw URL, to keep label cardinality bounded).
func NewMetricsMiddleware(metrics httpMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(recorder, r)

			path := chi.RouteContext(r.Context()).RoutePattern()
			if path == "" {
				path = "unmatched"
			}
			metrics.Observe(r.Method, path, recorder.status, time.Since(started))
		})
	}
}
