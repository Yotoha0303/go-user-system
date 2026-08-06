package handler

import (
	"context"
	"go-user-system/internal/model"
	"go-user-system/internal/response"
	"net/http"
	"testing"
)

type fakeRBACHandlerService struct {
	roleCodes       []string
	permissionCodes []string
	userID          int64
}

func (s *fakeRBACHandlerService) ListRoles(ctx context.Context) ([]model.Role, error) {
	return nil, nil
}

func (s *fakeRBACHandlerService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	return nil, nil
}

func (s *fakeRBACHandlerService) GetUserAuthorization(ctx context.Context, userID int64) ([]string, []string, error) {
	s.userID = userID
	return s.roleCodes, s.permissionCodes, nil
}

func (s *fakeRBACHandlerService) AssignRolesToUser(ctx context.Context, userID int64, roleCodes []string) error {
	return nil
}

func TestGetMyAuthorizationHandlerReturnsRoleAndPermissionCodes(t *testing.T) {
	fakeService := &fakeRBACHandlerService{
		roleCodes:       []string{"admin", "user"},
		permissionCodes: []string{"profile:read", "admin:roles:read"},
	}
	h := NewRBACHandler(fakeService)
	recorder := performJSONRequest(
		h.GetMyAuthorizationHandler,
		http.MethodGet,
		"/me/authorization",
		"",
		withUserID(9),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	body := decodeResponse(t, recorder)
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if fakeService.userID != 9 {
		t.Fatalf("expected user ID 9, got %d", fakeService.userID)
	}
	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", body.Data)
	}
	if len(data["role_codes"].([]interface{})) != 2 || len(data["permission_codes"].([]interface{})) != 2 {
		t.Fatalf("unexpected authorization data: %+v", data)
	}
}

func TestGetMyAuthorizationHandlerRejectsMissingUser(t *testing.T) {
	h := NewRBACHandler(&fakeRBACHandlerService{})
	recorder := performJSONRequest(h.GetMyAuthorizationHandler, http.MethodGet, "/me/authorization", "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	body := decodeResponse(t, recorder)
	if body.Code != response.CodeTokenUserMissing {
		t.Fatalf("expected code %d, got %d", response.CodeTokenUserMissing, body.Code)
	}
}
