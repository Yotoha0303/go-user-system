package router

import (
	"go-user-system/api"
	"go-user-system/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "pong"})
	})

	r.POST("/register", api.RegisterHandler)

	r.POST("/login", api.LoginHandler)
	return r
}
