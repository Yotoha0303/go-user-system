package router

import (
	"UsersSystem/utils"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "pong"})
	})
	return r
}
