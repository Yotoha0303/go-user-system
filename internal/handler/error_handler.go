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
		if appErr.Cause != nil {
			log.Printf("app error: path=%s method=%s code=%d msg=%s cause=%v",
				c.Request.URL.Path,
				c.Request.Method,
				appErr.Code,
				appErr.Message,
				appErr.Cause,
			)
		}
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
