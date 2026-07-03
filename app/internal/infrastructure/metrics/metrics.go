package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPMetrics implements the delivery httpMetrics port on top of Prometheus.
type HTTPMetrics struct {
	registry      *prometheus.Registry
	requestsTotal *prometheus.CounterVec
	errorsTotal   *prometheus.CounterVec
	duration      *prometheus.HistogramVec
}

func NewHTTPMetrics() *HTTPMetrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())

	requestsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Total number of HTTP requests that failed with a 5xx status.",
	}, []string{"method", "path", "status"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	registry.MustRegister(requestsTotal, errorsTotal, duration)

	return &HTTPMetrics{
		registry:      registry,
		requestsTotal: requestsTotal,
		errorsTotal:   errorsTotal,
		duration:      duration,
	}
}

// Observe records one served HTTP request.
func (m *HTTPMetrics) Observe(method, path string, status int, elapsed time.Duration) {
	statusLabel := strconv.Itoa(status)
	m.requestsTotal.WithLabelValues(method, path, statusLabel).Inc()
	if status >= http.StatusInternalServerError {
		m.errorsTotal.WithLabelValues(method, path, statusLabel).Inc()
	}
	m.duration.WithLabelValues(method, path).Observe(elapsed.Seconds())
}

// Handler exposes the /metrics scrape endpoint.
func (m *HTTPMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
