package router

import (
	"go-user-system/api"
	"go-user-system/middleware"
	"go-user-system/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	registerHealthRoutes(r)
	registerAPIRouter(r)

	return r
}

func registerHealthRoutes(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"message": "success",
		})
	})
}

func registerAPIRouter(rg *gin.Engine) {
	apiV1 := rg.Group("/api/v1")

	registerAuthRouter(apiV1)
	registerUsersRouter(apiV1)
}

func registerAuthRouter(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", api.RegisterHandler)
		auth.POST("/login", api.LoginHandler)

	}
}

func registerUsersRouter(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware())
	{
		users.GET("/me", api.MeHandler)
		users.PUT("/me/profile", api.UpdateProfileHandler)
	}
}
