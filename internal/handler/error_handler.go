package handler

import (
	"go-user-system/internal/apperror"
	"go-user-system/internal/response"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, err error, fallbackCode int, fallbackMessage string) {
	if appErr, ok := apperror.FromError(err); ok {
		response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}

	log.Printf("unexpected error: path=%s method=%s err=%v",
		c.Request.URL.Path,
		c.Request.Method,
		err,
	)
	response.Fail(c, http.StatusInternalServerError, fallbackCode, fallbackMessage)
}
