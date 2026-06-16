package router

import (
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/service"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, logger *slog.Logger, timeout time.Duration) *gin.Engine {
	r := gin.New()

	// TODO 新增自定义 Recovery 中间件，以捕获业务层报错
	r.Use(
		middleware.RequestID(),
		middleware.SlogMiddleware(logger),
		middleware.TimeoutMiddleware(timeout),
		gin.Recovery(),
	)

	userService := service.NewUserService(db)
	userHandler := handler.NewUserHandler(userService)
	healthHandler := handler.NewHealthHandler(db)

	registerHealthRoutes(r, healthHandler)
	registerAPIRoutes(r, userHandler)

	return r
}

func registerHealthRoutes(r *gin.Engine, healthHandler *handler.HealthHandler) {
	r.GET("/ping", healthHandler.PingHandler)
	r.GET("/livez", healthHandler.LivezHandler)
	r.GET("/readyz", healthHandler.ReadyzHandler)
}

func registerAPIRoutes(rg *gin.Engine, userHandler *handler.UserHandler) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRoutes(apiV1, userHandler)
	registerUsersRoutes(apiV1, userHandler)
}

func registerAuthRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", userHandler.RegisterHandler)
		auth.POST("/login", userHandler.LoginHandler)

	}
}

func registerUsersRoutes(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/me", userHandler.MeHandler)
		users.PUT("/me/profile", userHandler.UpdateProfileHandler)
		users.PUT("/me/login/out", userHandler.LoginOutHandler)
		users.PATCH("/me/update/password", userHandler.UpdateUserPasswordHandler)
	}
}
