package api

import (
	"go-user-system/service"
	"go-user-system/utils"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, 1001, "参数错误")
		return
	}

	err := service.Register(req.Username, req.Password)
	if err != nil {
		utils.Fail(c, 500, 1002, "注册失败")
		return
	}

	utils.Success(c, nil)
}
