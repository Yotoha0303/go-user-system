package observability

import (
	"go-user-system/internal/buildinfo"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const unmatchedRoute = "unmatched"

var standardMethods = map[string]struct{}{
	http.MethodConnect: {},
	http.MethodDelete:  {},
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
	http.MethodPatch:   {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodTrace:   {},
}

type Metrics struct {
	registry        *prometheus.Registry
	httpRequests    *prometheus.CounterVec
	httpDuration    *prometheus.HistogramVec
	httpRequestsNow *prometheus.GaugeVec
	readiness       prometheus.Gauge
}

func NewMetrics(build buildinfo.Info) *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry: registry,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "go_user_system",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests handled by method, route template, and status.",
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "go_user_system",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds by method and route template.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
		httpRequestsNow: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "go_user_system",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current in-flight HTTP requests by method.",
		}, []string{"method"}),
		readiness: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "go_user_system",
			Name:      "readiness",
			Help:      "Whether the most recent readiness check succeeded (1 ready, 0 not ready).",
		}),
	}
	buildMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "go_user_system",
		Name:      "build_info",
		Help:      "Build information for the running application.",
	}, []string{"version", "commit", "build_time"})
	buildMetric.WithLabelValues(build.Version, build.Commit, build.BuildTime).Set(1)

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		metrics.httpRequests,
		metrics.httpDuration,
		metrics.httpRequestsNow,
		metrics.readiness,
		buildMetric,
	)
	return metrics
}

func (m *Metrics) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		method := metricMethod(c.Request.Method)
		started := time.Now()
		m.httpRequestsNow.WithLabelValues(method).Inc()
		defer m.httpRequestsNow.WithLabelValues(method).Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = unmatchedRoute
		}
		m.httpRequests.WithLabelValues(method, route, strconv.Itoa(c.Writer.Status())).Inc()
		m.httpDuration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
	}
}

func metricMethod(method string) string {
	if _, ok := standardMethods[method]; ok {
		return method
	}
	return "OTHER"
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

func (m *Metrics) SetReady(ready bool) {
	if ready {
		m.readiness.Set(1)
		return
	}
	m.readiness.Set(0)
}
