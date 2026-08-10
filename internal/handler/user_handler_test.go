package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-user-system/internal/apperror"
	"go-user-system/internal/auth"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"go-user-system/internal/service"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type requestContextKey struct{}

type fakeUserService struct {
	registerErr           error
	loginUser             *model.User
	loginErr              error
	profileUser           *model.User
	profileErr            error
	updateErr             error
	updateUserPasswordErr error

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

func (s *fakeUserService) UpdateUserPassword(ctx context.Context, userID int64, req request.UpdatePasswordRequest) error {

	return s.updateUserPasswordErr
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

func testTokenManager(t *testing.T) *auth.TokenManager {
	t.Helper()

	tokenManager, err := auth.NewTokenManager(
		"handler_test_jwt_secret_32_chars",
		"go-user-system-test",
		24*time.Hour,
	)

	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}
	return tokenManager
}

func TestRegisterHandlerReturnsSuccess(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456789012"}`,
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
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456789012"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
	if body.Code != response.CodeUsernameAlreadyExists {
		t.Fatalf("expected code %d, got %d", response.CodeUsernameAlreadyExists, body.Code)
	}
}

func TestRegisterHandlerMapsWrappedAppErrorWithCause(t *testing.T) {
	fakeService := &fakeUserService{
		registerErr: apperror.Wrap(
			http.StatusInternalServerError,
			response.CodeRegisterFailed,
			"register failed",
			errors.New("insert failed"),
		),
	}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456789012"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeRegisterFailed {
		t.Fatalf("expected code %d, got %d", response.CodeRegisterFailed, body.Code)
	}
}

func TestRegisterHandlerMapsPlainErrorToFallback(t *testing.T) {
	fakeService := &fakeUserService{registerErr: errors.New("plain failure")}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.RegisterHandler,
		http.MethodPost,
		"/register",
		`{"username":"alice","password":"123456789012"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeRegisterFailed {
		t.Fatalf("expected code %d, got %d", response.CodeRegisterFailed, body.Code)
	}
}

func TestRegisterHandlerRejectsInvalidJSON(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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

func TestLoginHandlerRejectsInvalidJSON(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice"`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if body.Code != response.CodeInvalidParams {
		t.Fatalf("expected code %d, got %d", response.CodeInvalidParams, body.Code)
	}
	if fakeService.loginCtx != nil {
		t.Fatal("expected login service not to be called")
	}
}

func TestLoginHandlerReturnsTokenAndUser(t *testing.T) {
	testTokenManager(t)

	fakeService := &fakeUserService{
		loginUser: &model.User{
			ID:          1,
			Username:    "alice",
			Nickname:    "alice",
			Status:      model.UserStatusActive,
			AuthVersion: 1,
		},
	}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"123456789012"}`,
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
	if _, exists := data["refresh_token"]; exists {
		t.Fatal("expected refresh_token not to be exposed in response body")
	}

	var refreshCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == refreshTokenCookieName {
			refreshCookie = cookie
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected refresh token cookie")
	}
	if !refreshCookie.HttpOnly || refreshCookie.Path != refreshTokenCookiePath {
		t.Fatalf("unexpected refresh cookie attributes: %+v", refreshCookie)
	}
	if got := fakeService.loginCtx.Value(requestContextKey{}); got != "request-context" {
		t.Fatalf("expected request context to be passed to login service, got %v", got)
	}
}

type fakeAuthSessionService struct {
	rotatedOldJTI    string
	revokedJTI       string
	revokedAccessJTI string
	checkRetryAfter  time.Duration
	recordRetryAfter time.Duration
	checkErr         error
	recordErr        error
	resetErr         error
	checkedAccount   string
	checkedIP        string
	recordedAccount  string
	recordedIP       string
	resetAccount     string
}

func (s *fakeAuthSessionService) StoreRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	return nil
}

func (s *fakeAuthSessionService) RotateRefreshToken(ctx context.Context, userID int64, authVersion int64, oldJTI string, oldHash string, next *model.RefreshToken) error {
	s.rotatedOldJTI = oldJTI
	return nil
}

func (s *fakeAuthSessionService) RevokeRefreshToken(ctx context.Context, userID int64, jti string, tokenHash string) error {
	s.revokedJTI = jti
	return nil
}

func (s *fakeAuthSessionService) RevokeAccessToken(ctx context.Context, jti string, expiresAt time.Time) error {
	s.revokedAccessJTI = jti
	return nil
}

func (s *fakeAuthSessionService) CheckLoginAllowed(ctx context.Context, account, ip string) (time.Duration, error) {
	s.checkedAccount = account
	s.checkedIP = ip
	return s.checkRetryAfter, s.checkErr
}

func (s *fakeAuthSessionService) RecordLoginFailure(ctx context.Context, account, ip string) (time.Duration, error) {
	s.recordedAccount = account
	s.recordedIP = ip
	return s.recordRetryAfter, s.recordErr
}

func (s *fakeAuthSessionService) ResetLoginFailures(ctx context.Context, account string) error {
	s.resetAccount = account
	return s.resetErr
}

func TestRefreshTokenHandlerReadsCookieAndRotatesCookie(t *testing.T) {
	manager := testTokenManager(t)
	issuedToken, err := manager.GenerateRefreshToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}
	sessionService := &fakeAuthSessionService{}
	userHandler := NewUserHandler(&fakeUserService{}, manager, sessionService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/refresh", userHandler.RefreshTokenHandler)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: issuedToken.Token})
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if sessionService.rotatedOldJTI != issuedToken.JTI {
		t.Fatalf("expected old JTI %q, got %q", issuedToken.JTI, sessionService.rotatedOldJTI)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != refreshTokenCookieName || !cookies[0].HttpOnly {
		t.Fatalf("expected rotated HttpOnly refresh cookie, got %+v", cookies)
	}
}

func TestLogoutHandlerReadsAndClearsCookie(t *testing.T) {
	manager := testTokenManager(t)
	issuedToken, err := manager.GenerateRefreshToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}
	sessionService := &fakeAuthSessionService{}
	userHandler := NewUserHandler(&fakeUserService{}, manager, sessionService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/logout", userHandler.LogoutHandler)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: issuedToken.Token})
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if sessionService.revokedJTI != issuedToken.JTI {
		t.Fatalf("expected revoked JTI %q, got %q", issuedToken.JTI, sessionService.revokedJTI)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != refreshTokenCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("expected cleared refresh cookie, got %+v", cookies)
	}
}

func TestLogoutHandlerRevokesPresentedAccessTokenJTI(t *testing.T) {
	manager := testTokenManager(t)
	refreshToken, err := manager.GenerateRefreshToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}
	accessToken, err := manager.GenerateAccessTokenIssue(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}
	sessionService := &fakeAuthSessionService{}
	userHandler := NewUserHandler(&fakeUserService{}, manager, sessionService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/logout", userHandler.LogoutHandler)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: refreshToken.Token})
	req.Header.Set("Authorization", "Bearer "+accessToken.Token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if sessionService.revokedAccessJTI != accessToken.JTI {
		t.Fatalf("expected access JTI %q revoked, got %q", accessToken.JTI, sessionService.revokedAccessJTI)
	}
}

func TestLoginHandlerReturnsRateLimitWithRetryAfter(t *testing.T) {
	fakeService := &fakeUserService{}
	sessionService := &fakeAuthSessionService{checkRetryAfter: 90 * time.Second}
	userHandler := NewUserHandler(fakeService, testTokenManager(t), sessionService)

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"wrong-password"}`,
	)
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusTooManyRequests || body.Code != response.CodeLoginRateLimited {
		t.Fatalf("expected rate limit response, status=%d code=%d", recorder.Code, body.Code)
	}
	if recorder.Header().Get("Retry-After") != "90" {
		t.Fatalf("expected Retry-After 90, got %q", recorder.Header().Get("Retry-After"))
	}
	if fakeService.loginCtx != nil {
		t.Fatal("expected password verification skipped while limited")
	}
}

func TestLoginHandlerRecordsInvalidCredentialsAndLimitsThresholdAttempt(t *testing.T) {
	fakeService := &fakeUserService{loginErr: service.ErrInvalidCredentials}
	sessionService := &fakeAuthSessionService{recordRetryAfter: time.Minute}
	userHandler := NewUserHandler(fakeService, testTokenManager(t), sessionService)

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"wrong-password"}`,
	)
	body := decodeResponse(t, recorder)

	if recorder.Code != http.StatusTooManyRequests || body.Code != response.CodeLoginRateLimited {
		t.Fatalf("expected threshold attempt rate limited, status=%d code=%d", recorder.Code, body.Code)
	}
	if sessionService.recordedAccount != "alice" || sessionService.recordedIP == "" {
		t.Fatalf("expected account and IP failure recorded, got account=%q ip=%q", sessionService.recordedAccount, sessionService.recordedIP)
	}
}

func TestLoginHandlerClearsAccountFailuresAfterSuccess(t *testing.T) {
	fakeService := &fakeUserService{loginUser: &model.User{
		ID:          7,
		Username:    "alice",
		Nickname:    "alice",
		Status:      model.UserStatusActive,
		AuthVersion: 1,
	}}
	sessionService := &fakeAuthSessionService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t), sessionService)

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"password1234"}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if sessionService.resetAccount != "alice" {
		t.Fatalf("expected alice failure counter reset, got %q", sessionService.resetAccount)
	}
}

func TestLoginHandlerMapsTokenGenerationError(t *testing.T) {
	fakeService := &fakeUserService{
		loginUser: &model.User{
			ID:          1,
			Username:    "alice",
			Nickname:    "alice",
			Status:      model.UserStatusActive,
			AuthVersion: 1,
		},
	}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))
	userHandler.generateToken = func(userID int64, username string, authVersion int64) (string, error) {
		return "", errors.New("sign failed")
	}

	recorder := performJSONRequest(
		userHandler.LoginHandler,
		http.MethodPost,
		"/login",
		`{"username":"alice","password":"123456789012"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeTokenGenerateFailed {
		t.Fatalf("expected code %d, got %d", response.CodeTokenGenerateFailed, body.Code)
	}
}

func TestLoginHandlerMapsInvalidCredentials(t *testing.T) {
	fakeService := &fakeUserService{loginErr: service.ErrInvalidCredentials}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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

func TestLoginHandlerUsesForwardedIPOnlyFromTrustedProxy(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies []string
		forwardedFor   string
		expectedIP     string
	}{
		{name: "trusted proxy", trustedProxies: []string{"192.0.2.10"}, forwardedFor: "198.51.100.25", expectedIP: "198.51.100.25"},
		{name: "spoofed prefix", trustedProxies: []string{"192.0.2.10"}, forwardedFor: "203.0.113.66, 198.51.100.25", expectedIP: "198.51.100.25"},
		{name: "untrusted proxy", trustedProxies: nil, forwardedFor: "198.51.100.25", expectedIP: "192.0.2.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := &fakeUserService{loginErr: service.ErrInvalidCredentials}
			sessionService := &fakeAuthSessionService{}
			userHandler := NewUserHandler(fakeService, testTokenManager(t), sessionService)
			gin.SetMode(gin.TestMode)
			router := gin.New()
			if err := router.SetTrustedProxies(tt.trustedProxies); err != nil {
				t.Fatalf("set trusted proxies failed: %v", err)
			}
			router.POST("/login", userHandler.LoginHandler)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"alice","password":"password1234"}`))
			req.RemoteAddr = "192.0.2.10:12345"
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			router.ServeHTTP(recorder, req)

			if sessionService.checkedIP != tt.expectedIP {
				t.Fatalf("expected checked IP %q, got %q", tt.expectedIP, sessionService.checkedIP)
			}
		})
	}
}

func TestUserHandlerUsesExplicitSecureCookieSetting(t *testing.T) {
	issuedToken := &auth.IssuedToken{
		Token:     "refresh-token",
		ExpiresAt: time.Now().Add(time.Hour),
		ExpiresIn: int64(time.Hour.Seconds()),
	}

	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprintf("secure=%t", secure), func(t *testing.T) {
			handler := NewUserHandlerWithOptions(
				&fakeUserService{},
				testTokenManager(t),
				UserHandlerOptions{SecureCookies: secure},
			)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/login", nil)
			context.Request.Header.Set("X-Forwarded-Proto", "https")

			handler.setRefreshTokenCookie(context, issuedToken)
			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Secure != secure {
				t.Fatalf("expected Secure=%t, got %+v", secure, cookies)
			}
		})
	}
}

func TestMeHandlerMapsServiceError(t *testing.T) {
	fakeService := &fakeUserService{profileErr: service.ErrUserNotFound}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.MeHandler,
		http.MethodGet,
		"/me",
		"",
		withUserID(1),
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	if body.Code != response.CodeUserNotFound {
		t.Fatalf("expected code %d, got %d", response.CodeUserNotFound, body.Code)
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
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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

func TestUpdateProfileHandlerRejectsMissingUserID(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.UpdateProfileHandler,
		http.MethodPut,
		"/me/profile",
		`{"nickname":"new_name"}`,
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if body.Code != response.CodeTokenUserMissing {
		t.Fatalf("expected code %d, got %d", response.CodeTokenUserMissing, body.Code)
	}
	if fakeService.updateCalled {
		t.Fatal("expected update service not to be called")
	}
}

func TestUpdateProfileHandlerRejectsInvalidUserIDType(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.UpdateProfileHandler,
		http.MethodPut,
		"/me/profile",
		`{"nickname":"new_name"}`,
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
	if fakeService.updateCalled {
		t.Fatal("expected update service not to be called")
	}
}

func TestUpdateProfileHandlerCallsService(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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

func TestUpdateProfileHandlerMapsServiceError(t *testing.T) {
	fakeService := &fakeUserService{updateErr: service.ErrNicknameTooLong}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

	recorder := performJSONRequest(
		userHandler.UpdateProfileHandler,
		http.MethodPut,
		"/me/profile",
		`{"nickname":"too-long"}`,
		withUserID(7),
	)

	body := decodeResponse(t, recorder)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if body.Code != response.CodeNicknameInvalid {
		t.Fatalf("expected code %d, got %d", response.CodeNicknameInvalid, body.Code)
	}
}

func TestUpdateProfileHandlerRejectsInvalidJSON(t *testing.T) {
	fakeService := &fakeUserService{}
	userHandler := NewUserHandler(fakeService, testTokenManager(t))

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
