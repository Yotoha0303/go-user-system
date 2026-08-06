package router

import (
	"go-user-system/internal/auth"
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/model"
	"go-user-system/internal/service"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager) *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(logger),
		middleware.Recovery(logger),
	)

	userService := service.NewUserService(db)
	authService := service.NewAuthService(db)
	rbacService := service.NewRBACService(db)
	userHandler := handler.NewUserHandler(userService, tokenManager, authService)
	rbacHandler := handler.NewRBACHandler(rbacService)
	healthHandler := handler.NewHealthHandler(db)

	registerHealthRoutes(r, healthHandler)
	registerSwaggerRoutes(r)
	registerAPIRoutes(r, userHandler, rbacHandler, tokenManager, rbacService)

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
	rbacService *service.RBACService,
) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRoutes(apiV1, userHandler)
	registerUsersRoutes(apiV1, userHandler, rbacHandler, tokenManager, rbacService)
	registerAdminRoutes(apiV1, rbacHandler, tokenManager, rbacService)
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
	rbacService *service.RBACService,
) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(tokenManager))
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
	rbacService *service.RBACService,
) {
	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware(tokenManager))
	{
		admin.GET("/roles", middleware.RequirePermission(rbacService, model.PermissionAdminRolesRead), rbacHandler.ListRolesHandler)
		admin.GET("/permissions", middleware.RequirePermission(rbacService, model.PermissionAdminPermsRead), rbacHandler.ListPermissionsHandler)
		admin.PUT("/users/:id/roles", middleware.RequirePermission(rbacService, model.PermissionAdminUserRoleEdit), rbacHandler.AssignUserRolesHandler)
	}
}
