package router

import (
	"go-user-system/api"
	"go-user-system/middleware"
	"go-user-system/response"

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
		auth.POST("/register", api.RegisterHandler)
		auth.POST("/login", api.LoginHandler)

	}
}

func registerUsersRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/me", api.MeHandler)
		users.PUT("/me/profile", api.UpdateProfileHandler)
	}
}
