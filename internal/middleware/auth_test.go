package middleware

import (
	"encoding/json"
	"go-user-system/internal/auth"
	"go-user-system/internal/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func performAuthRequest(authHeader string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(&auth.TokenManager{}), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		response.Success(c, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		request.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(recorder, request)

	return recorder
}

func decodeAuthResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	return body
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	recorder := performAuthRequest("")
	body := decodeAuthResponse(t, recorder)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Code != response.CodeTokenMissing {
		t.Fatalf("expected code %d, got %d", response.CodeTokenMissing, body.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidFormat(t *testing.T) {
	recorder := performAuthRequest("token-value")
	body := decodeAuthResponse(t, recorder)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Code != response.CodeTokenInvalidFormat {
		t.Fatalf("expected code %d, got %d", response.CodeTokenInvalidFormat, body.Code)
	}
}

func TestAuthMiddlewareRejectsEmptyBearerToken(t *testing.T) {
	recorder := performAuthRequest("Bearer ")
	body := decodeAuthResponse(t, recorder)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Code != response.CodeTokenInvalidFormat {
		t.Fatalf("expected code %d, got %d", response.CodeTokenInvalidFormat, body.Code)
	}
}
