package router

import (
	"go-user-system/api"
	"go-user-system/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	apiV1 := r.Group("/api/v1")
	{
		apiV1.POST("/register", api.RegisterHandler)

		apiV1.POST("/login", api.LoginHandler)

		authGrop := apiV1.Group("/")
		authGrop.Use(middleware.AuthMiddleware())
		{
			authGrop.GET("/me", api.MeHandler)
		}
	}

	return r
}
