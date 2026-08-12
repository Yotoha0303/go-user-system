package router

import (
	"fmt"
	"go-user-system/internal/auth"
	"go-user-system/internal/authstate"
	"go-user-system/internal/buildinfo"
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/model"
	"go-user-system/internal/observability"
	"go-user-system/internal/service"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthRuntime struct {
	StateStore          authstate.Store
	LoginRateLimit      service.LoginRateLimit
	RegistrationEnabled *bool
	SecureCookies       bool
	TrustedProxies      []string
	RequestTimeout      time.Duration
}

func SetupRouter(db *gorm.DB, logger *slog.Logger, tokenManager *auth.TokenManager, runtimes ...AuthRuntime) http.Handler {
	r := gin.New()
	build := buildinfo.Current()
	metrics := observability.NewMetrics(build)
	runtime := AuthRuntime{}
	if len(runtimes) > 0 {
		runtime = runtimes[0]
	}
	if err := r.SetTrustedProxies(runtime.TrustedProxies); err != nil {
		panic(fmt.Sprintf("configure trusted proxies: %v", err))
	}

	r.Use(
		middleware.RequestID(),
		metrics.RouteMiddleware(),
		middleware.AccessLog(logger),
		middleware.Recovery(logger),
	)

	userService := service.NewUserService(db)
	authService := service.NewAuthService(db)
	var healthCheckers []handler.HealthChecker
	if runtime.StateStore != nil {
		authService = service.NewAuthServiceWithState(db, runtime.StateStore, runtime.LoginRateLimit)
		healthCheckers = append(healthCheckers, runtime.StateStore)
	}
	rbacService := service.NewRBACService(db)
	userHandler := handler.NewUserHandlerWithOptions(userService, tokenManager, handler.UserHandlerOptions{
		SecureCookies: runtime.SecureCookies,
	}, authService)
	rbacHandler := handler.NewRBACHandler(rbacService)
	healthHandler := handler.NewHealthHandlerWithRecorder(db, metrics, healthCheckers...)
	systemHandler := handler.NewSystemHandler(build)

	registerSystemRoutes(r, healthHandler, systemHandler, metrics)
	registerSwaggerRoutes(r)
	registrationEnabled := true
	if runtime.RegistrationEnabled != nil {
		registrationEnabled = *runtime.RegistrationEnabled
	}
	registerAPIRoutes(r, userHandler, rbacHandler, tokenManager, authService, rbacService, registrationEnabled)

	return metrics.HTTPHandler(middleware.TimeoutHandler(r, runtime.RequestTimeout))
}

func registerSystemRoutes(r *gin.Engine, healthHandler *handler.HealthHandler, systemHandler *handler.SystemHandler, metrics *observability.Metrics) {
	r.GET("/ping", healthHandler.PingHandler)
	r.GET("/livez", healthHandler.LivezHandler)
	r.GET("/readyz", healthHandler.ReadyzHandler)
	r.GET("/version", systemHandler.VersionHandler)
	r.GET("/metrics", gin.WrapH(metrics.Handler()))
}

func registerAPIRoutes(
	rg *gin.Engine,
	userHandler *handler.UserHandler,
	rbacHandler *handler.RBACHandler,
	tokenManager *auth.TokenManager,
	authService *service.AuthService,
	rbacService *service.RBACService,
	registrationEnabled bool,
) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRoutes(apiV1, userHandler, registrationEnabled)
	registerUsersRoutes(apiV1, userHandler, rbacHandler, tokenManager, authService, rbacService)
	registerAdminRoutes(apiV1, rbacHandler, tokenManager, authService, rbacService)
}

func registerAuthRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler, registrationEnabled bool) {
	auth := rg.Group("/auth")
	{
		if registrationEnabled {
			auth.POST("/register", userHandler.RegisterHandler)
		}
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
