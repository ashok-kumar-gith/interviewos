package server

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics bundles the Prometheus collectors for HTTP server instrumentation,
// registered against a dedicated registry so the exposition is isolated from
// the process-global default registry and is safe to construct per-engine
// (e.g. in tests) without duplicate-registration panics.
type Metrics struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewMetrics builds the HTTP metrics collectors and registers them, along with
// the Go runtime and process collectors, onto a fresh registry.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, labeled by method, route template, and status code.",
		},
		[]string{"method", "path", "status"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, labeled by method, route template, and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	reg.MustRegister(
		requestsTotal,
		requestDuration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return &Metrics{
		registry:        reg,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
	}
}

// Middleware records request counts and latencies. It uses gin's route template
// (c.FullPath()) rather than the raw URL path to bound label cardinality;
// unmatched routes (404s) are bucketed under "" -> "<unmatched>".
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		m.requestsTotal.WithLabelValues(method, path, status).Inc()
		m.requestDuration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	}
}

// Handler returns the Prometheus exposition HTTP handler bound to this
// Metrics registry (Prometheus text format).
func (m *Metrics) Handler() gin.HandlerFunc {
	h := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	return gin.WrapH(h)
}
