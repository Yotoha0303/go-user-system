package router

import (
	"go-user-system/internal/handler"
	"go-user-system/internal/middleware"
	"go-user-system/internal/response"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	registerHealthRoutes(r)
	registerAPIRoutes(r)

	return r
}

func registerHealthRoutes(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		response.Success(c, gin.H{
			"message": "success",
		})
	})
}

func registerAPIRoutes(rg *gin.Engine) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRoutes(apiV1)
	registerUsersRoutes(apiV1)
}

func registerAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", handler.RegisterHandler)
		auth.POST("/login", handler.LoginHandler)

	}
}

func registerUsersRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/me", handler.MeHandler)
		users.PUT("/me/profile", handler.UpdateProfileHandler)
	}
}
