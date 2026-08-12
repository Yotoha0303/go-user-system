package router

import (
	"encoding/json"
	"go-user-system/internal/auth"
	"go-user-system/internal/response"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPingReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)

	SetupRouter(nil, testLogger(), &auth.TokenManager{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected business code %d, got %d", response.CodeSuccess, body.Code)
	}
}

func TestReadyzFailsWhenDatabaseIsNotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	SetupRouter(nil, testLogger(), &auth.TokenManager{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != response.CodeReadinessFailed {
		t.Fatalf("expected business code %d, got %d", response.CodeReadinessFailed, body.Code)
	}
}

func TestRegistrationRouteCanBeDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	disabled := false
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)

	SetupRouter(nil, testLogger(), &auth.TokenManager{}, AuthRuntime{
		RegistrationEnabled: &disabled,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestSystemRoutesExposeVersionAndMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := SetupRouter(nil, testLogger(), &auth.TokenManager{})

	for _, path := range []string{"/version", "/metrics"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		r.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s status %d, got %d", path, http.StatusOK, recorder.Code)
		}
	}
}
