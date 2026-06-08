package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func performResponseTestRequest(handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(recorder, request)

	return recorder
}

func decodeTestResponse(t *testing.T, recorder *httptest.ResponseRecorder) Response {
	t.Helper()

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return body
}

func TestSuccessWritesStandardSuccessResponse(t *testing.T) {
	recorder := performResponseTestRequest(func(c *gin.Context) {
		Success(c, gin.H{"id": 1})
	})

	body := decodeTestResponse(t, recorder)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != CodeSuccess {
		t.Fatalf("expected code %d, got %d", CodeSuccess, body.Code)
	}
	if body.Msg != "success" {
		t.Fatalf("expected message success, got %s", body.Msg)
	}
	if body.Data == nil {
		t.Fatal("expected response data")
	}
}

func TestFailWritesStandardFailureResponse(t *testing.T) {
	recorder := performResponseTestRequest(func(c *gin.Context) {
		Fail(c, http.StatusBadRequest, CodeInvalidParams, "invalid params")
	})

	body := decodeTestResponse(t, recorder)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if body.Code != CodeInvalidParams {
		t.Fatalf("expected code %d, got %d", CodeInvalidParams, body.Code)
	}
	if body.Msg != "invalid params" {
		t.Fatalf("expected message invalid params, got %s", body.Msg)
	}
	if body.Data != nil {
		t.Fatalf("expected nil data, got %v", body.Data)
	}
}
