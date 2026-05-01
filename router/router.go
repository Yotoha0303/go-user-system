package router

import (
	"go-user-system/api"
	"go-user-system/middleware"
	"go-user-system/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	//测试接口
	r.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"message": "success",
		})
	})

	apiV1 := r.Group("/api/v1")
	{
		apiV1.POST("/register", api.RegisterHandler)

		apiV1.POST("/login", api.LoginHandler)

		authGrop := apiV1.Group("")
		authGrop.Use(middleware.AuthMiddleware())
		{
			authGrop.GET("/me", api.MeHandler)
		}
	}

	return r
}
