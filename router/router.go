package router

import (
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/model"
	"go-user-system/internal/service"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthRuntime struct {
	StateStore     authstate.Store
	LoginRateLimit service.LoginRateLimit
}

func SetupRouter(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtimes ...AuthRuntime) *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(logger),
		middleware.Recovery(logger),
	)

	userService := service.NewUserService(db)
	authService := service.NewAuthService(db)
	var healthCheckers []handler.HealthChecker
	if len(runtimes) > 0 && runtimes[0].StateStore != nil {
		authService = service.NewAuthServiceWithState(db, runtimes[0].StateStore, runtimes[0].LoginRateLimit)
		healthCheckers = append(healthCheckers, runtimes[0].StateStore)
	}
	rbacService := service.NewRBACService(db)
	userHandler := handler.NewUserHandler(userService, tokenManager, authService)
	rbacHandler := handler.NewRBACHandler(rbacService)
	healthHandler := handler.NewHealthHandler(db, healthCheckers...)

	registerHealthRoutes(r, healthHandler)
	registerSwaggerRoutes(r)
	registerAPIRoutes(r, userHandler, rbacHandler, tokenManager, authService, rbacService)

	return r
}

func registerHealthRoutes(r *gin.Engine, healthHandler *handler.HealthHandler) {
	r.GET("/ping", healthHandler.PingHandler)
	r.GET("/livez", healthHandler.LivezHandler)
	r.GET("/readyz", healthHandler.ReadyzHandler)
}

func registerAPIRoutes(
	rg *gin.Engine,
	userHandler *handler.UserHandler,
	rbacHandler *handler.RBACHandler,
	tokenManager *auth.TokenManager,
	authService *service.AuthService,
	rbacService *service.RBACService,
) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRoutes(apiV1, userHandler)
	registerUsersRoutes(apiV1, userHandler, rbacHandler, tokenManager, authService, rbacService)
	registerAdminRoutes(apiV1, rbacHandler, tokenManager, authService, rbacService)
}

func registerAuthRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", userHandler.RegisterHandler)
		auth.POST("/login", userHandler.LoginHandler)
		auth.POST("/refresh", userHandler.RefreshTokenHandler)
		auth.POST("/logout", userHandler.LogoutHandler)

	}
}

func registerUsersRoutes(
	rg *gin.RouterGroup,
	userHandler *handler.UserHandler,
	rbacHandler *handler.RBACHandler,
	tokenManager *auth.TokenManager,
	authService *service.AuthService,
	rbacService *service.RBACService,
) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(tokenManager, authService))
	{
		users.GET("/me", middleware.RequirePermission(rbacService, model.PermissionProfileRead), userHandler.MeHandler)
		users.GET("/me/authorization", rbacHandler.GetMyAuthorizationHandler)
		users.PUT("/me/profile", middleware.RequirePermission(rbacService, model.PermissionProfileUpdate), userHandler.UpdateProfileHandler)
		users.PATCH("/me/update/password", middleware.RequirePermission(rbacService, model.PermissionPasswordUpdate), userHandler.UpdateUserPasswordHandler)
	}
}

func registerAdminRoutes(
	rg *gin.RouterGroup,
	rbacHandler *handler.RBACHandler,
	tokenManager *auth.TokenManager,
	authService *service.AuthService,
	rbacService *service.RBACService,
) {
	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware(tokenManager, authService))
	{
		admin.GET("/roles", middleware.RequirePermission(rbacService, model.PermissionAdminRolesRead), rbacHandler.ListRolesHandler)
		admin.GET("/permissions", middleware.RequirePermission(rbacService, model.PermissionAdminPermsRead), rbacHandler.ListPermissionsHandler)
		admin.PUT("/users/:id/roles", middleware.RequirePermission(rbacService, model.PermissionAdminUserRoleEdit), rbacHandler.AssignUserRolesHandler)
	}
}
