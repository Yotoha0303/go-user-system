package handler

import (
	"go-user-system/internal/buildinfo"
	"go-user-system/internal/response"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	build buildinfo.Info
}

func NewSystemHandler(build buildinfo.Info) *SystemHandler {
	return &SystemHandler{build: build}
}

// VersionHandler godoc
// @Summary 查询运行版本
// @Tags system
// @Produce json
// @Success 200 {object} response.Response
// @Router /version [get]
func (h *SystemHandler) VersionHandler(c *gin.Context) {
	response.Success(c, h.build)
}
