package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"go-user-system/internal/apperror"
	"go-user-system/internal/dao"
	"go-user-system/internal/model"
	"go-user-system/internal/request"
	"go-user-system/internal/response"
	"go-user-system/internal/testutil"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const serviceSQLDriverName = "go_user_system_service_test"

var registerServiceSQLDriverOnce sync.Once

type serviceSQLDriver struct{}

func (serviceSQLDriver) Open(name string) (driver.Conn, error) {
	return serviceSQLConn{}, nil
}

type serviceSQLConn struct{}

func (serviceSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (serviceSQLConn) Close() error {
	return nil
}

func (serviceSQLConn) Begin() (driver.Tx, error) {
	return serviceSQLTx{}, nil
}

func (serviceSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (serviceSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return serviceSQLRows{}, nil
}

type serviceSQLTx struct{}

func (serviceSQLTx) Commit() error {
	return nil
}

func (serviceSQLTx) Rollback() error {
	return nil
}

type serviceSQLRows struct{}

func (serviceSQLRows) Columns() []string {
	return []string{"id", "username", "password_hash", "nickname", "status", "created_at", "updated_at", "last_login_at", "deleted_at"}
}

func (serviceSQLRows) Close() error {
	return nil
}

func (serviceSQLRows) Next(dest []driver.Value) error {
	return io.EOF
}

func openServiceDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	registerServiceSQLDriverOnce.Do(func() {
		sql.Register(serviceSQLDriverName, serviceSQLDriver{})
	})

	sqlDB, err := sql.Open(serviceSQLDriverName, "service")
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db failed: %v", err)
		}
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dry run db failed: %v", err)
	}
	return db
}

type fakeUserStore struct {
	userByUsername     *model.User
	userByUsernameErr  error
	userByID           *model.User
	userByIDErr        error
	createErr          error
	updateNicknameErr  error
	updateLastLoginErr error

	createdUser                   *model.User
	updatedUserID                 int64
	updatedNickname               string
	lastLoginUserID               int64
	lastLoginAt                   time.Time
	createCalled                  bool
	updateCalled                  bool
	lastLoginCalled               bool
	getUsernameInput              string
	updateUserPasswordByUserIDErr error
	oldPasswordHash               string
	newPasswordHash               string
}

type fakeRBACRepo struct {
	assignedRoleCodes []string
	assignErr         error
}

func (s *fakeUserStore) CreateUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	s.createCalled = true
	s.createdUser = user
	return s.createErr
}

func (s *fakeUserStore) GetUserByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error) {
	s.getUsernameInput = username
	return s.userByUsername, s.userByUsernameErr
}

func (s *fakeUserStore) GetUserByID(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return s.userByID, s.userByIDErr
}

func (s *fakeUserStore) GetUserByIDForUpdate(ctx context.Context, db *gorm.DB, id int64) (*model.User, error) {
	return s.userByID, s.userByIDErr
}

func (s *fakeUserStore) UpdateNicknameByID(ctx context.Context, db *gorm.DB, id int64, nickname string) error {
	s.updateCalled = true
	s.updatedUserID = id
	s.updatedNickname = nickname
	return s.updateNicknameErr
}

func (s *fakeUserStore) UpdateLastLoginAtByID(ctx context.Context, db *gorm.DB, id int64, lastLoginAt time.Time) error {
	s.lastLoginCalled = true
	s.lastLoginUserID = id
	s.lastLoginAt = lastLoginAt
	return s.updateLastLoginErr
}

func (s *fakeUserStore) UpdateUserPasswordByUserID(ctx context.Context, db *gorm.DB, userID int64, oldPasswordHash string, newPasswordHash string) error {
	s.lastLoginUserID = userID
	s.oldPasswordHash = oldPasswordHash
	s.newPasswordHash = newPasswordHash
	return s.updateUserPasswordByUserIDErr
}

// TODO
func (s *fakeUserStore) ListUser(ctx context.Context, db *gorm.DB, limit, offset int) (model.User, error) {
	var user model.User
	return user, nil
}

// TODO
func (s *fakeUserStore) UserDisabled(ctx context.Context, db *gorm.DB, userID int64) error {
	return nil
}

func (r *fakeRBACRepo) AssignRoleToUserByCode(ctx context.Context, db *gorm.DB, userID int64, roleCode string) error {
	if r.assignErr != nil {
		return r.assignErr
	}
	r.assignedRoleCodes = append(r.assignedRoleCodes, roleCode)
	return nil
}

func (r *fakeRBACRepo) UserHasPermission(ctx context.Context, db *gorm.DB, userID int64, permissionCode string) (bool, error) {
	return false, nil
}

func (r *fakeRBACRepo) ListRoles(ctx context.Context, db *gorm.DB) ([]model.Role, error) {
	return nil, nil
}

func (r *fakeRBACRepo) ListPermissions(ctx context.Context, db *gorm.DB) ([]model.Permission, error) {
	return nil, nil
}

func (r *fakeRBACRepo) ListUserRoleCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error) {
	return nil, nil
}

func (r *fakeRBACRepo) ListUserPermissionCodes(ctx context.Context, db *gorm.DB, userID int64) ([]string, error) {
	return nil, nil
}

func (r *fakeRBACRepo) ReplaceUserRolesByCodes(ctx context.Context, db *gorm.DB, userID int64, roleCodes []string) error {
	return nil
}

func newUnitUserService(store *fakeUserStore) *UserService {
	return &UserService{
		db:    &gorm.DB{},
		store: store,
	}
}

func assertServiceAppError(t *testing.T, err error, httpStatus int, code int) {
	t.Helper()

	appErr, ok := apperror.FromError(err)
	if !ok {
		t.Fatalf("expected AppError, got %T %v", err, err)
	}
	if appErr.HTTPStatus != httpStatus {
		t.Fatalf("expected http status %d, got %d", httpStatus, appErr.HTTPStatus)
	}
	if appErr.Code != code {
		t.Fatalf("expected code %d, got %d", code, appErr.Code)
	}
}

func passwordHash(t *testing.T, password string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash failed: %v", err)
	}
	return string(hash)
}

func TestEnsureDBRejectsNilService(t *testing.T) {
	var userService *UserService

	err := userService.ensureDB()

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestDaoUserStoreDelegatesToDAO(t *testing.T) {
	store := daoUserStore{}
	db := openServiceDryRunDB(t)
	ctx := context.Background()
	user := &model.User{
		Username:     "alice",
		PasswordHash: "hash",
		Nickname:     "alice",
		Status:       model.UserStatusActive,
	}

	if err := store.CreateUser(ctx, db, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if _, err := store.GetUserByUsername(ctx, db, "alice"); err != nil {
		t.Fatalf("get user by username failed: %v", err)
	}
	if _, err := store.GetUserByID(ctx, db, 1); err != nil {
		t.Fatalf("get user by id failed: %v", err)
	}
	if err := store.UpdateNicknameByID(ctx, db, 1, "new-name"); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	if err := store.UpdateLastLoginAtByID(ctx, db, 1, time.Now()); err != nil {
		t.Fatalf("update last login failed: %v", err)
	}
}

func TestRegisterValidatesUsernameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "ab",
		Password: "123456",
	})

	if !errors.Is(err, ErrUsernameTooShort) {
		t.Fatalf("expected ErrUsernameTooShort, got %v", err)
	}
}

func TestRegisterValidatesPasswordLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "12345",
	})

	if !errors.Is(err, ErrPasswordTooShortOrTooLong) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestValidatePasswordBoundaries(t *testing.T) {
	if err := validatePassword("12345678901"); !errors.Is(err, ErrPasswordTooShortOrTooLong) {
		t.Fatalf("expected 11-character password rejection, got %v", err)
	}
	if err := validatePassword("123456789012"); err != nil {
		t.Fatalf("expected 12-character password acceptance, got %v", err)
	}
	if err := validatePassword(strings.Repeat("a", MaxPasswordBytes+1)); !errors.Is(err, ErrPasswordTooShortOrTooLong) {
		t.Fatalf("expected password over %d bytes rejection, got %v", MaxPasswordBytes, err)
	}
}

func TestRegisterRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "123456789012",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestLoginRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "123456",
	})

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestGetProfileValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(context.Background(), 0)

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestGetProfileRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	_, err := userService.GetProfile(context.Background(), 1)

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestUpdateNicknameValidatesUserID(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 0, "alice")

	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestUpdateNicknameValidatesEmptyNickname(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, "   ")

	if !errors.Is(err, ErrNicknameEmpty) {
		t.Fatalf("expected ErrNicknameEmpty, got %v", err)
	}
}

func TestUpdateNicknameValidatesNicknameLength(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, strings.Repeat("a", 65))

	if !errors.Is(err, ErrNicknameTooLong) {
		t.Fatalf("expected ErrNicknameTooLong, got %v", err)
	}
}

func TestUpdateNicknameRequiresDatabase(t *testing.T) {
	userService := NewUserService(nil)

	err := userService.UpdateNickname(context.Background(), 1, "alice")

	if !errors.Is(err, ErrDatabaseNotInitialized) {
		t.Fatalf("expected ErrDatabaseNotInitialized, got %v", err)
	}
}

func TestRegisterCreatesActiveUserWithTrimmedUsernameAndHashedPassword(t *testing.T) {
	store := &fakeUserStore{userByUsernameErr: gorm.ErrRecordNotFound}
	userService := newUnitUserService(store)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "  alice  ",
		Password: "password1234",
	})

	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if store.getUsernameInput != "alice" {
		t.Fatalf("expected trimmed username alice, got %s", store.getUsernameInput)
	}
	if !store.createCalled {
		t.Fatal("expected CreateUser to be called")
	}
	if store.createdUser.Username != "alice" {
		t.Fatalf("expected username alice, got %s", store.createdUser.Username)
	}
	if store.createdUser.Nickname != "alice" {
		t.Fatalf("expected nickname alice, got %s", store.createdUser.Nickname)
	}
	if store.createdUser.Status != model.UserStatusActive {
		t.Fatalf("expected active status, got %d", store.createdUser.Status)
	}
	if store.createdUser.PasswordHash == "password1234" {
		t.Fatal("expected password hash, got plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.createdUser.PasswordHash), []byte("password1234")); err != nil {
		t.Fatalf("password hash does not match: %v", err)
	}
}

func TestRegisterAssignsUserRoleToRegularUser(t *testing.T) {
	store := &fakeUserStore{
		userByUsernameErr: gorm.ErrRecordNotFound,
	}
	rbacRepo := &fakeRBACRepo{}
	userService := newUnitUserService(store)
	userService.db = openServiceDryRunDB(t)
	userService.rbacRepo = rbacRepo

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "password1234",
	})

	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if len(rbacRepo.assignedRoleCodes) != 1 || rbacRepo.assignedRoleCodes[0] != model.RoleCodeUser {
		t.Fatalf("expected only user role, got %v", rbacRepo.assignedRoleCodes)
	}
}

func TestRegisterAssignsOnlyUserRoleToFirstUser(t *testing.T) {
	store := &fakeUserStore{
		userByUsernameErr: gorm.ErrRecordNotFound,
	}
	rbacRepo := &fakeRBACRepo{}
	userService := newUnitUserService(store)
	userService.db = openServiceDryRunDB(t)
	userService.rbacRepo = rbacRepo

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "password1234",
	})

	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if len(rbacRepo.assignedRoleCodes) != 1 || rbacRepo.assignedRoleCodes[0] != model.RoleCodeUser {
		t.Fatalf("expected only user role, got %v", rbacRepo.assignedRoleCodes)
	}
}

func TestRegisterRejectsExistingUsername(t *testing.T) {
	store := &fakeUserStore{userByUsername: &model.User{ID: 1, Username: "alice"}}
	userService := newUnitUserService(store)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "password1234",
	})

	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Fatalf("expected ErrUsernameAlreadyExists, got %v", err)
	}
}

func TestRegisterWrapsUsernameLookupError(t *testing.T) {
	store := &fakeUserStore{userByUsernameErr: errors.New("lookup failed")}
	userService := newUnitUserService(store)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "password1234",
	})

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeRegisterFailed)
}

func TestRegisterRejectsPasswordOverBcryptLimit(t *testing.T) {
	store := &fakeUserStore{userByUsernameErr: gorm.ErrRecordNotFound}
	userService := newUnitUserService(store)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: strings.Repeat("a", MaxPasswordBytes+1),
	})

	assertServiceAppError(t, err, http.StatusBadRequest, response.CodeInvalidParams)
}

func TestRegisterWrapsCreateUserError(t *testing.T) {
	store := &fakeUserStore{
		userByUsernameErr: gorm.ErrRecordNotFound,
		createErr:         errors.New("insert failed"),
	}
	userService := newUnitUserService(store)

	err := userService.Register(context.Background(), request.RegisterRequest{
		Username: "alice",
		Password: "password1234",
	})

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeRegisterFailed)
}

func TestLoginMapsMissingUserToInvalidCredentials(t *testing.T) {
	store := &fakeUserStore{userByUsernameErr: gorm.ErrRecordNotFound}
	userService := newUnitUserService(store)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "password1234",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginWrapsUsernameLookupError(t *testing.T) {
	store := &fakeUserStore{userByUsernameErr: errors.New("select failed")}
	userService := newUnitUserService(store)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "password1234",
	})

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeLoginFailed)
}

func TestLoginMapsDisabledUserToInvalidCredentials(t *testing.T) {
	store := &fakeUserStore{
		userByUsername: &model.User{
			ID:           1,
			Username:     "alice",
			PasswordHash: passwordHash(t, "password1234"),
			Status:       model.UserStatusDisabled,
		},
	}
	userService := newUnitUserService(store)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "password1234",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	store := &fakeUserStore{
		userByUsername: &model.User{
			ID:           1,
			Username:     "alice",
			PasswordHash: passwordHash(t, "password1234"),
			Status:       model.UserStatusActive,
		},
	}
	userService := newUnitUserService(store)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "wrong-password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginWrapsLastLoginUpdateError(t *testing.T) {
	store := &fakeUserStore{
		userByUsername: &model.User{
			ID:           1,
			Username:     "alice",
			PasswordHash: passwordHash(t, "password1234"),
			Status:       model.UserStatusActive,
		},
		updateLastLoginErr: errors.New("update failed"),
	}
	userService := newUnitUserService(store)

	_, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "alice",
		Password: "password1234",
	})

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeLoginFailed)
}

func TestLoginReturnsUserAndUpdatesLastLogin(t *testing.T) {
	store := &fakeUserStore{
		userByUsername: &model.User{
			ID:           1,
			Username:     "alice",
			PasswordHash: passwordHash(t, "password1234"),
			Status:       model.UserStatusActive,
		},
	}
	userService := newUnitUserService(store)

	user, err := userService.Login(context.Background(), request.LoginRequest{
		Username: "  alice  ",
		Password: "password1234",
	})

	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if user.LastLoginAt == nil || user.LastLoginAt.IsZero() {
		t.Fatal("expected LastLoginAt to be set")
	}
	if !store.lastLoginCalled {
		t.Fatal("expected UpdateLastLoginAtByID to be called")
	}
	if store.lastLoginUserID != 1 {
		t.Fatalf("expected last login user id 1, got %d", store.lastLoginUserID)
	}
	if store.lastLoginAt.IsZero() {
		t.Fatal("expected stored last login timestamp")
	}
}

func TestGetProfileMapsMissingUserToNotFound(t *testing.T) {
	store := &fakeUserStore{userByIDErr: gorm.ErrRecordNotFound}
	userService := newUnitUserService(store)

	_, err := userService.GetProfile(context.Background(), 1)

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetProfileWrapsLookupError(t *testing.T) {
	store := &fakeUserStore{userByIDErr: errors.New("select failed")}
	userService := newUnitUserService(store)

	_, err := userService.GetProfile(context.Background(), 1)

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeGetProfileFailed)
}

func TestGetProfileRejectsDisabledUser(t *testing.T) {
	store := &fakeUserStore{userByID: &model.User{ID: 1, Status: model.UserStatusDisabled}}
	userService := newUnitUserService(store)

	_, err := userService.GetProfile(context.Background(), 1)

	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled, got %v", err)
	}
}

func TestGetProfileReturnsActiveUser(t *testing.T) {
	store := &fakeUserStore{userByID: &model.User{ID: 1, Username: "alice", Status: model.UserStatusActive}}
	userService := newUnitUserService(store)

	user, err := userService.GetProfile(context.Background(), 1)

	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("expected username alice, got %s", user.Username)
	}
}

func TestUpdateNicknameReturnsNilWhenNicknameIsUnchanged(t *testing.T) {
	store := &fakeUserStore{userByID: &model.User{ID: 1, Nickname: "alice", Status: model.UserStatusActive}}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, " alice ")

	if err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	if store.updateCalled {
		t.Fatal("expected unchanged nickname not to call update")
	}
}

func TestUpdateNicknameMapsMissingUserToNotFound(t *testing.T) {
	store := &fakeUserStore{userByIDErr: gorm.ErrRecordNotFound}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, "new-name")

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateNicknameWrapsLookupError(t *testing.T) {
	store := &fakeUserStore{userByIDErr: errors.New("select failed")}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, "new-name")

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeUpdateNicknameFailed)
}

func TestUpdateNicknameRejectsDisabledUser(t *testing.T) {
	store := &fakeUserStore{userByID: &model.User{ID: 1, Nickname: "alice", Status: model.UserStatusDisabled}}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, "new-name")

	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled, got %v", err)
	}
}

func TestUpdateNicknameWrapsUpdateError(t *testing.T) {
	store := &fakeUserStore{
		userByID:          &model.User{ID: 1, Nickname: "alice", Status: model.UserStatusActive},
		updateNicknameErr: errors.New("update failed"),
	}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, "new-name")

	assertServiceAppError(t, err, http.StatusInternalServerError, response.CodeUpdateNicknameFailed)
}

func TestUpdateNicknameTrimsAndUpdatesNickname(t *testing.T) {
	store := &fakeUserStore{userByID: &model.User{ID: 1, Nickname: "alice", Status: model.UserStatusActive}}
	userService := newUnitUserService(store)

	err := userService.UpdateNickname(context.Background(), 1, "  new-name  ")

	if err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	if !store.updateCalled {
		t.Fatal("expected UpdateNicknameByID to be called")
	}
	if store.updatedUserID != 1 {
		t.Fatalf("expected user id 1, got %d", store.updatedUserID)
	}
	if store.updatedNickname != "new-name" {
		t.Fatalf("expected trimmed nickname new-name, got %s", store.updatedNickname)
	}
}

func TestUpdatePasswordChangesHashAndRevokesRefreshTokensInTransaction(t *testing.T) {
	oldHash := passwordHash(t, "old-password")
	store := &fakeUserStore{userByID: &model.User{
		ID:           7,
		PasswordHash: oldHash,
		Status:       model.UserStatusActive,
		AuthVersion:  1,
	}}
	repo := &fakeRefreshTokenRepo{current: &model.RefreshToken{UserID: 7}}
	userService := &UserService{
		db:          openServiceDryRunDB(t),
		store:       store,
		refreshRepo: repo,
	}

	err := userService.UpdateUserPassword(context.Background(), 7, request.UpdatePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err != nil {
		t.Fatalf("update password failed: %v", err)
	}
	if store.oldPasswordHash != oldHash {
		t.Fatal("expected password update to be guarded by old hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.newPasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("expected new password hash, got %v", err)
	}
	if repo.current.RevokedAt == nil || repo.current.RevokedReason == nil || *repo.current.RevokedReason != model.RefreshTokenRevokedReasonPasswordChange {
		t.Fatalf("expected refresh tokens revoked for password change, got %+v", repo.current)
	}
}

func prepareUserServiceIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testutil.OpenMySQL(t)
	testutil.ResetTables(t, db, "user_roles", "roles", "schema_migrations", "users")
	testutil.CreateUsersTable(t, db)
	testutil.CreateRoleAssignmentTables(t, db)

	t.Cleanup(func() {
		testutil.ResetTables(t, db, "user_roles", "roles", "schema_migrations", "users")
		testutil.CloseMySQL(t, db)
	})

	return db
}

func TestUserServiceIntegrationRegisterLoginProfileAndNickname(t *testing.T) {
	db := prepareUserServiceIntegrationDB(t)
	userService := NewUserService(db)
	ctx := context.Background()
	username := testutil.UniqueName(t, "svc_user")
	password := "password1234"

	err := userService.Register(ctx, request.RegisterRequest{
		Username: "  " + username + "  ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	storedUser, err := dao.GetUserByUsername(ctx, db, username)
	if err != nil {
		t.Fatalf("get registered user failed: %v", err)
	}
	if storedUser.PasswordHash == password {
		t.Fatal("expected password to be stored as hash, got plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
		t.Fatalf("stored password hash does not match password: %v", err)
	}
	if storedUser.Nickname != username {
		t.Fatalf("expected nickname %s, got %s", username, storedUser.Nickname)
	}
	if storedUser.Status != model.UserStatusActive {
		t.Fatalf("expected active status, got %d", storedUser.Status)
	}
	var roleCodes []string
	if err := db.Table("user_roles AS ur").
		Joins("JOIN roles AS r ON r.id = ur.role_id").
		Where("ur.user_id = ?", storedUser.ID).
		Order("r.code ASC").
		Pluck("r.code", &roleCodes).Error; err != nil {
		t.Fatalf("list registered user roles failed: %v", err)
	}
	if len(roleCodes) != 1 || roleCodes[0] != model.RoleCodeUser {
		t.Fatalf("expected registered user role [user], got %v", roleCodes)
	}

	err = userService.Register(ctx, request.RegisterRequest{
		Username: username,
		Password: password,
	})
	if !errors.Is(err, ErrUsernameAlreadyExists) {
		t.Fatalf("expected ErrUsernameAlreadyExists, got %v", err)
	}

	_, err = userService.Login(ctx, request.LoginRequest{
		Username: username,
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	loggedInUser, err := userService.Login(ctx, request.LoginRequest{
		Username: "  " + username + "  ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loggedInUser.LastLoginAt == nil || loggedInUser.LastLoginAt.IsZero() {
		t.Fatal("expected login response to include last_login_at")
	}

	afterLoginUser, err := dao.GetUserByID(ctx, db, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get user after login failed: %v", err)
	}
	if afterLoginUser.LastLoginAt == nil || afterLoginUser.LastLoginAt.IsZero() {
		t.Fatal("expected last_login_at to be persisted")
	}

	profileUser, err := userService.GetProfile(ctx, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if profileUser.Username != username {
		t.Fatalf("expected profile username %s, got %s", username, profileUser.Username)
	}

	if err := userService.UpdateNickname(ctx, loggedInUser.ID, "  new-nickname  "); err != nil {
		t.Fatalf("update nickname failed: %v", err)
	}
	updatedUser, err := dao.GetUserByID(ctx, db, loggedInUser.ID)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}
	if updatedUser.Nickname != "new-nickname" {
		t.Fatalf("expected nickname new-nickname, got %s", updatedUser.Nickname)
	}
}

func TestUserServiceIntegrationRejectsDisabledUser(t *testing.T) {
	db := prepareUserServiceIntegrationDB(t)
	userService := NewUserService(db)
	ctx := context.Background()
	username := testutil.UniqueName(t, "disabled_user")
	password := "password1234"

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash failed: %v", err)
	}

	disabledUser := model.User{
		Username:     username,
		PasswordHash: string(hashBytes),
		Nickname:     username,
		Status:       model.UserStatusDisabled,
	}
	if err := dao.CreateUser(ctx, db, &disabledUser); err != nil {
		t.Fatalf("create disabled user failed: %v", err)
	}

	_, err = userService.Login(ctx, request.LoginRequest{
		Username: username,
		Password: password,
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on login, got %v", err)
	}

	_, err = userService.GetProfile(ctx, disabledUser.ID)
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled on profile, got %v", err)
	}

	err = userService.UpdateNickname(ctx, disabledUser.ID, "new-nickname")
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled on nickname update, got %v", err)
	}
}
