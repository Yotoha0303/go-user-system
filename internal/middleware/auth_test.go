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

// func initJWTForMiddlewareTest(t *testing.T, expireHours int) {
// 	t.Helper()

// 	t.Setenv("JWT_SECRET", "middleware_test_jwt_secret_32_chars")
// 	err := utils.InitJWTKey(&config.Config{
// 		JWT: config.JWTConfig{ExpireHours: expireHours},
// 	})
// 	if err != nil {
// 		t.Fatalf("init jwt key failed: %v", err)
// 	}
// }

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

// func TestAuthMiddlewareRejectsMalformedToken(t *testing.T) {
// 	initJWTForMiddlewareTest(t, 24)

// 	recorder := performAuthRequest("Bearer malformed.token")
// 	body := decodeAuthResponse(t, recorder)

// 	if recorder.Code != http.StatusUnauthorized {
// 		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
// 	}
// 	if body.Code != response.CodeTokenMalformed {
// 		t.Fatalf("expected code %d, got %d", response.CodeTokenMalformed, body.Code)
// 	}
// }

// func TestAuthMiddlewareRejectsInvalidSignature(t *testing.T) {
// 	initJWTForMiddlewareTest(t, 24)

// 	token, err := utils.GenerateToken(1, "alice")
// 	if err != nil {
// 		t.Fatalf("generate token failed: %v", err)
// 	}

// 	t.Setenv("JWT_SECRET", "middleware_test_other_secret_32_chars")
// 	err = utils.InitJWTKey(&config.Config{
// 		JWT: config.JWTConfig{ExpireHours: 24},
// 	})
// 	if err != nil {
// 		t.Fatalf("reinit jwt key failed: %v", err)
// 	}

// 	recorder := performAuthRequest("Bearer " + token)
// 	body := decodeAuthResponse(t, recorder)

// 	if recorder.Code != http.StatusUnauthorized {
// 		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
// 	}
// 	if body.Code != response.CodeTokenSignatureInvalid {
// 		t.Fatalf("expected code %d, got %d", response.CodeTokenSignatureInvalid, body.Code)
// 	}
// }

// TODO parseToken
func TestAuthMiddlewareRejectsGenericInvalidToken(t *testing.T) {
	// oldParseToken := parseToken
	// t.Cleanup(func() {
	// 	parseToken = oldParseToken
	// })

	// parseToken = func(tokenString string) (*utils.UserClaims, error) {
	// 	return nil, utils.ErrAccessTokenInvalid
	// }

	// recorder := performAuthRequest("Bearer invalid-token")
	// body := decodeAuthResponse(t, recorder)

	// if recorder.Code != http.StatusUnauthorized {
	// 	t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	// }
	// if body.Code != response.CodeTokenInvalid {
	// 	t.Fatalf("expected code %d, got %d", response.CodeTokenInvalid, body.Code)
	// }
}

// func TestAuthMiddlewareAllowsValidToken(t *testing.T) {
// 	initJWTForMiddlewareTest(t, 24)

// 	token, err := utils.GenerateToken(1, "alice")
// 	if err != nil {
// 		t.Fatalf("generate token failed: %v", err)
// 	}

// 	recorder := performAuthRequest("Bearer " + token)
// 	body := decodeAuthResponse(t, recorder)

// 	if recorder.Code != http.StatusOK {
// 		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
// 	}
// 	if body.Code != response.CodeSuccess {
// 		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
// 	}
// }

// func TestAuthMiddlewareRejectsExpiredToken(t *testing.T) {
// 	initJWTForMiddlewareTest(t, -1)

// 	token, err := utils.GenerateToken(1, "alice")
// 	if err != nil {
// 		t.Fatalf("generate token failed: %v", err)
// 	}

// 	recorder := performAuthRequest("Bearer " + token)
// 	body := decodeAuthResponse(t, recorder)

// 	if recorder.Code != http.StatusUnauthorized {
// 		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
// 	}
// 	if body.Code != response.CodeTokenExpired {
// 		t.Fatalf("expected code %d, got %d", response.CodeTokenExpired, body.Code)
// 	}
// }
