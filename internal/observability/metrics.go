package observability

import (
	"context"
	"go-user-system/internal/buildinfo"
	"net/http"
	"strconv"
	"sync/atomic"
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

type routeState struct {
	value atomic.Value
}

type routeStateContextKey struct{}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
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

func (m *Metrics) RouteMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if state, ok := c.Request.Context().Value(routeStateContextKey{}).(*routeState); ok {
			route := c.FullPath()
			if route == "" {
				route = unmatchedRoute
			}
			state.value.Store(route)
		}
		c.Next()
	}
}

func (m *Metrics) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		state := &routeState{}
		state.value.Store(unmatchedRoute)
		r = r.WithContext(context.WithValue(r.Context(), routeStateContextKey{}, state))

		method := metricMethod(r.Method)
		started := time.Now()
		m.httpRequestsNow.WithLabelValues(method).Inc()
		defer m.httpRequestsNow.WithLabelValues(method).Dec()

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		route := state.value.Load().(string)
		m.httpRequests.WithLabelValues(method, route, strconv.Itoa(recorder.status)).Inc()
		m.httpDuration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
	})
}

func metricMethod(method string) string {
	if _, ok := standardMethods[method]; ok {
		return method
	}
	return "OTHER"
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
