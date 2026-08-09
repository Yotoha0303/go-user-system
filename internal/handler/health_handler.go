package handler

import (
	"context"
	"go-user-system/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db       *gorm.DB
	checkers []HealthChecker
}

func NewHealthHandler(db *gorm.DB, checkers ...HealthChecker) *HealthHandler {
	return &HealthHandler{db: db, checkers: checkers}
}

// PingHandler godoc
// @Summary 基础连通性检查
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Router /ping [get]
func (h *HealthHandler) PingHandler(c *gin.Context) {
	response.Success(c, gin.H{
		"message": "success",
	})
}

// LivezHandler godoc
// @Summary 应用存活检查
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Router /livez [get]
func (h *HealthHandler) LivezHandler(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "alive",
	})
}

// ReadyzHandler godoc
// @Summary 服务就绪检查
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Failure 503 {object} response.Response
// @Router /readyz [get]
func (h *HealthHandler) ReadyzHandler(c *gin.Context) {
	if h.db == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not initialized")
		return
	}

	if h.db.Config == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not ready")
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not ready")
		return
	}

	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not ready")
		return
	}

	for _, checker := range h.checkers {
		if checker == nil {
			continue
		}
		if err := checker.Ping(c.Request.Context()); err != nil {
			response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "authentication state store is not ready")
			return
		}
	}

	response.Success(c, gin.H{
		"status": "ready",
	})
}
