package utils

import (
	"go-user-system/model"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, model.Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func Fail(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, model.Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}
