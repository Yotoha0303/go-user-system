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

	// err := service.Register(req.Username, req.Password)
	// if err != nil {
	// 	utils.Fail(c, 500, 1002, "注册失败")
	// 	return
	// }

	if err := service.Register(req.Username, req.Password); err != nil {
		switch err.Error() {
		case "username is empty", "password is empty", "username too short", "password too short":
			utils.Fail(c, 400, 1001, err.Error())
			return
		case "username already exists":
			utils.Fail(c, 400, 1002, err.Error())
			return
		default:
			utils.Fail(c, 500, 1003, "register failed")
			return
		}
	}

	utils.Success(c, nil)
}
