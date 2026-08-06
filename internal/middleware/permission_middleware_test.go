package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"go-user-system/internal/response"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePermissionChecker struct {
	allowed bool
	err     error
	userID  int64
	code    string
}

func (c *fakePermissionChecker) HasPermission(ctx context.Context, userID int64, permissionCode string) (bool, error) {
	c.userID = userID
	c.code = permissionCode
	return c.allowed, c.err
}

func performPermissionRequest(checker *fakePermissionChecker, userID interface{}) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/protected",
		func(c *gin.Context) {
			if userID != nil {
				c.Set("user_id", userID)
			}
			c.Next()
		},
		RequirePermission(checker, "profile:read"),
		func(c *gin.Context) {
			response.Success(c, gin.H{"ok": true})
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodePermissionResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return body
}

func TestRequirePermissionAllowsAuthorizedUser(t *testing.T) {
	checker := &fakePermissionChecker{allowed: true}

	recorder := performPermissionRequest(checker, int64(7))
	body := decodePermissionResponse(t, recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if checker.userID != 7 || checker.code != "profile:read" {
		t.Fatalf("unexpected checker args: userID=%d code=%s", checker.userID, checker.code)
	}
}

func TestRequirePermissionRejectsUnauthorizedUser(t *testing.T) {
	checker := &fakePermissionChecker{allowed: false}

	recorder := performPermissionRequest(checker, int64(7))
	body := decodePermissionResponse(t, recorder)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
	if body.Code != response.CodePermissionDenied {
		t.Fatalf("expected code %d, got %d", response.CodePermissionDenied, body.Code)
	}
}

func TestRequirePermissionRejectsMissingUser(t *testing.T) {
	checker := &fakePermissionChecker{allowed: true}

	recorder := performPermissionRequest(checker, nil)
	body := decodePermissionResponse(t, recorder)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Code != response.CodeTokenUserMissing {
		t.Fatalf("expected code %d, got %d", response.CodeTokenUserMissing, body.Code)
	}
}

func TestRequirePermissionMapsCheckerError(t *testing.T) {
	checker := &fakePermissionChecker{err: errors.New("query failed")}

	recorder := performPermissionRequest(checker, int64(7))
	body := decodePermissionResponse(t, recorder)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeRBACFailed {
		t.Fatalf("expected code %d, got %d", response.CodeRBACFailed, body.Code)
	}
}
