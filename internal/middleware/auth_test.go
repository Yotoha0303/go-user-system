package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"go-user-system/internal/auth"
	"go-user-system/internal/response"
	"go-user-system/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeAccessTokenValidator struct {
	err    error
	claims *auth.UserClaims
}

func (v *fakeAccessTokenValidator) ValidateAccessToken(ctx context.Context, claims *auth.UserClaims) error {
	v.claims = claims
	return v.err
}

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

func TestAuthMiddlewareValidatesDatabaseBackedSession(t *testing.T) {
	manager, err := auth.NewTokenManagerWithTTL(
		"middleware_test_jwt_secret_32_chars",
		"middleware-test",
		time.Minute,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}
	token, err := manager.GenerateAccessToken(7, "alice", 3)
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}
	validator := &fakeAccessTokenValidator{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", AuthMiddleware(manager, validator), func(c *gin.Context) {
		response.Success(c, nil)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if validator.claims == nil || validator.claims.AuthVersion != 3 {
		t.Fatalf("expected auth version claims passed to validator, got %+v", validator.claims)
	}
}

func TestAuthMiddlewareRejectsInvalidSession(t *testing.T) {
	manager, err := auth.NewTokenManagerWithTTL(
		"middleware_test_jwt_secret_32_chars",
		"middleware-test",
		time.Minute,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}
	token, err := manager.GenerateAccessToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}
	validator := &fakeAccessTokenValidator{err: service.ErrAccessSessionInvalid}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", AuthMiddleware(manager, validator), func(c *gin.Context) {
		t.Fatal("protected handler must not run")
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !errors.Is(validator.err, service.ErrAccessSessionInvalid) {
		t.Fatalf("expected invalid session error, got %v", validator.err)
	}
}
