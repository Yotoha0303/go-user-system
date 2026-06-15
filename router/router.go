package router

import (
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/service"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, logger *slog.Logger) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.SlogMiddleware(logger))

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
	}
}
