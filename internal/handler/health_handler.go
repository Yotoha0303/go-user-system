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

type ReadinessRecorder interface {
	SetReady(ready bool)
}

type HealthHandler struct {
	db                *gorm.DB
	checkers          []HealthChecker
	readinessRecorder ReadinessRecorder
}

func NewHealthHandler(db *gorm.DB, checkers ...HealthChecker) *HealthHandler {
	return NewHealthHandlerWithRecorder(db, nil, checkers...)
}

func NewHealthHandlerWithRecorder(db *gorm.DB, recorder ReadinessRecorder, checkers ...HealthChecker) *HealthHandler {
	return &HealthHandler{db: db, checkers: checkers, readinessRecorder: recorder}
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
		h.failReadiness(c, "database is not initialized")
		return
	}

	if h.db.Config == nil {
		h.failReadiness(c, "database is not ready")
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		h.failReadiness(c, "database is not ready")
		return
	}

	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		h.failReadiness(c, "database is not ready")
		return
	}

	for _, checker := range h.checkers {
		if checker == nil {
			continue
		}
		if err := checker.Ping(c.Request.Context()); err != nil {
			h.failReadiness(c, "authentication state store is not ready")
			return
		}
	}

	h.recordReadiness(true)
	response.Success(c, gin.H{
		"status": "ready",
	})
}

func (h *HealthHandler) failReadiness(c *gin.Context, message string) {
	h.recordReadiness(false)
	response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, message)
}

func (h *HealthHandler) recordReadiness(ready bool) {
	if h.readinessRecorder != nil {
		h.readinessRecorder.SetReady(ready)
	}
}
