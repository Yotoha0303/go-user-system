package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"go-user-system/config"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"go-user-system/internal/service"
	"go-user-system/internal/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type requestContextKey struct{}

type fakeUserService struct {
	registerErr error
	loginUser   *model.User
	loginErr    error
	profileUser *model.User
	profileErr  error
	updateErr   error

	registerCalled bool
	updateCalled   bool
	updatedUserID  int64
	updatedName    string
	registerCtx    context.Context
	loginCtx       context.Context
	profileCtx     context.Context
	updateCtx      context.Context
}

func (s *fakeUserService) Register(ctx context.Context, req request.RegisterRequest) error {
	s.registerCalled = true
	s.registerCtx = ctx
	return s.registerErr
}

func (s *fakeUserService) Login(ctx context.Context, req request.LoginRequest) (*model.User, error) {
	s.loginCtx = ctx
	return s.loginUser, s.loginErr
}

func (s *fakeUserService) GetProfile(ctx context.Context, userID int64) (*model.User, error) {
	s.profileCtx = ctx
	return s.profileUser, s.profileErr
}

func (s *fakeUserService) UpdateNickname(ctx context.Context, userID int64, nickname string) error {
	s.updateCalled = true
	s.updateCtx = ctx
	s.updatedUserID = userID
	s.updatedName = nickname
	return s.updateErr
}

func performJSONRequest(handlerFunc gin.HandlerFunc, method string, path string, body string, middlewares ...gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handlers := append(middlewares, handlerFunc)
	router.Handle(method, path, handlers...)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request = request.WithContext(context.WithValue(request.Context(), requestContextKey{}, "request-context"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var body response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	return body
}

func withUserID(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func initJWTForHandlerTest(t *testing.T) {
	t.Helper()

	t.Setenv("JWT_SECRET", "handler_test_jwt_secret_32_chars")
	err := utils.InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})
	if err != nil {
		t.Fatalf("init jwt key failed: %v", err)
	}
}

func TestRegisterHandlerReturnsSuccess(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if !fakeService.registerCalled {
		t.Fatal("expected register service to be called")
	}
	if got := fakeService.registerCtx.Value(requestContextKey{}); got != "request-context" {
		t.Fatalf("expected request context to be passed to service, got %v", got)
	}
}

func TestRegisterHandlerMapsServiceError(t *testing.T) {
	fakeService := &fakeUserService{registerErr: service.ErrUsernameAlreadyExists}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
	if body.Code != response.CodeUsernameAlreadyExists {
		t.Fatalf("expected code %d, got %d", response.CodeUsernameAlreadyExists, body.Code)
	}
}

func TestRegisterHandlerRejectsInvalidJSON(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice"`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if body.Code != response.CodeInvalidParams {
		t.Fatalf("expected code %d, got %d", response.CodeInvalidParams, body.Code)
	}
	if fakeService.registerCalled {
		t.Fatal("expected register service not to be called")
	}
}

func TestLoginHandlerReturnsTokenAndUser(t *testing.T) {
	initJWTForHandlerTest(t)

	fakeService := &fakeUserService{
		loginUser: &model.User{
			ID:       1,
			Username: "alice",
			Nickname: "alice",
			Status:   model.UserStatusActive,
		},
	}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"123456"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", body.Data)
	}
	if data["access_token"] == "" {
		t.Fatal("expected access_token to be returned")
	}
	if got := fakeService.loginCtx.Value(requestContextKey{}); got != "request-context" {
		t.Fatalf("expected request context to be passed to login service, got %v", got)
	}
}

func TestLoginHandlerMapsInvalidCredentials(t *testing.T) {
	fakeService := &fakeUserService{loginErr: service.ErrInvalidCredentials}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"wrong-password"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Code != response.CodeLoginFailed {
		t.Fatalf("expected code %d, got %d", response.CodeLoginFailed, body.Code)
	}
}

func TestMeHandlerReturnsCurrentUser(t *testing.T) {
	fakeService := &fakeUserService{
		profileUser: &model.User{
			ID:       1,
			Username: "alice",
			Nickname: "alice",
			Status:   model.UserStatusActive,
		},
	}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.MeHandler,
		http.MethodGet,
		"/me",
		"",
		withUserID(1),
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if got := fakeService.profileCtx.Value(requestContextKey{}); got != "request-context" {
		t.Fatalf("expected request context to be passed to profile service, got %v", got)
	}
}

func TestMeHandlerRejectsMissingUserID(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.MeHandler,
		http.MethodGet,
		"/me",
		"",
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeTokenUserMissing {
		t.Fatalf("expected code %d, got %d", response.CodeTokenUserMissing, body.Code)
	}
}

func TestMeHandlerRejectsInvalidUserIDType(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.MeHandler,
		http.MethodGet,
		"/me",
		"",
		func(c *gin.Context) {
			c.Set("user_id", "bad-user-id")
			c.Next()
		},
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeTokenUserInvalid {
		t.Fatalf("expected code %d, got %d", response.CodeTokenUserInvalid, body.Code)
	}
}

func TestUpdateProfileHandlerCallsService(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.UpdateProfileHandler,
		http.MethodPut,
		"/me/profile",
		`{"nickname":"new_name"}`,
		withUserID(7),
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if body.Code != response.CodeSuccess {
		t.Fatalf("expected code %d, got %d", response.CodeSuccess, body.Code)
	}
	if !fakeService.updateCalled {
		t.Fatal("expected update service to be called")
	}
	if fakeService.updatedUserID != 7 || fakeService.updatedName != "new_name" {
		t.Fatalf("unexpected update args: userID=%d nickname=%s", fakeService.updatedUserID, fakeService.updatedName)
	}
	if got := fakeService.updateCtx.Value(requestContextKey{}); got != "request-context" {
		t.Fatalf("expected request context to be passed to update service, got %v", got)
	}
}

func TestUpdateProfileHandlerRejectsInvalidJSON(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService)

	recorder := performJSONRequest(
		userHandler.UpdateProfileHandler,
		http.MethodPut,
		"/me/profile",
		`{"nickname":`,
		withUserID(1),
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if body.Code != response.CodeInvalidParams {
		t.Fatalf("expected code %d, got %d", response.CodeInvalidParams, body.Code)
	}
	if fakeService.updateCalled {
		t.Fatal("expected update service not to be called")
	}
}
