package observability

import (
	"go-user-system/internal/buildinfo"
	"go-user-system/internal/middleware"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMetricsRecordsRouteTemplatesAndReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewMetrics(buildinfo.Info{
		Version:   "v1.2.3",
		Commit:    "abc123",
		BuildTime: "2026-08-12T10:00:00Z",
	})
	metrics.SetReady(true)

	r := gin.New()
	r.Use(metrics.RouteMiddleware())
	r.GET("/users/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	requestRecorder := httptest.NewRecorder()
	handler := metrics.HTTPHandler(r)
	handler.ServeHTTP(requestRecorder, httptest.NewRequest(http.MethodGet, "/users/42", nil))
	if requestRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, requestRecorder.Code)
	}

	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRecorder.Body.String()

	expectedSamples := []string{
		`go_user_system_build_info{build_time="2026-08-12T10:00:00Z",commit="abc123",version="v1.2.3"} 1`,
		`go_user_system_http_requests_total{method="GET",route="/users/:id",status="204"} 1`,
		`go_user_system_readiness 1`,
	}
	for _, sample := range expectedSamples {
		if !strings.Contains(body, sample) {
			t.Fatalf("expected metrics body to contain %q\n%s", sample, body)
		}
	}
	if strings.Contains(body, "/users/42") {
		t.Fatalf("expected metrics to use route template, got raw path\n%s", body)
	}
}

func TestMetricsUsesBoundedLabelForUnmatchedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewMetrics(buildinfo.Current())
	r := gin.New()
	r.Use(metrics.RouteMiddleware())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	handler := metrics.HTTPHandler(r)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing/123", nil))
	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if !strings.Contains(metricsRecorder.Body.String(), `route="unmatched"`) {
		t.Fatalf("expected unmatched route label\n%s", metricsRecorder.Body.String())
	}
}

func TestMetricsUsesBoundedLabelForCustomMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewMetrics(buildinfo.Current())
	r := gin.New()
	r.Use(metrics.RouteMiddleware())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	handler := metrics.HTTPHandler(r)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("UNTRUSTED-METHOD", "/missing", nil))
	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := metricsRecorder.Body.String()
	if !strings.Contains(body, `method="OTHER"`) || strings.Contains(body, `method="UNTRUSTED-METHOD"`) {
		t.Fatalf("expected custom method to use OTHER label\n%s", body)
	}
}

func TestMetricsRecordsClientFacingTimeoutOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewMetrics(buildinfo.Current())
	r := gin.New()
	r.Use(metrics.RouteMiddleware())
	releaseHandler := make(chan struct{})
	r.GET("/slow", func(c *gin.Context) {
		<-releaseHandler
		c.Status(http.StatusNoContent)
	})
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	handler := metrics.HTTPHandler(middleware.TimeoutHandler(r, 5*time.Millisecond))
	timeoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(timeoutRecorder, httptest.NewRequest(http.MethodGet, "/slow", nil))
	if timeoutRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected client-facing status %d, got %d", http.StatusServiceUnavailable, timeoutRecorder.Code)
	}

	close(releaseHandler)
	time.Sleep(10 * time.Millisecond)
	metricsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRecorder.Body.String()
	expected := `go_user_system_http_requests_total{method="GET",route="/slow",status="503"} 1`
	if !strings.Contains(body, expected) {
		t.Fatalf("expected timeout metric %q\n%s", expected, body)
	}
	if strings.Contains(body, `route="/slow",status="204"`) {
		t.Fatalf("expected timeout to be recorded only once with client-facing status\n%s", body)
	}
	if !strings.Contains(body, `go_user_system_http_requests_in_flight{method="GET"} 0`) {
		t.Fatalf("expected in-flight request to return to zero after timeout\n%s", body)
	}
}
