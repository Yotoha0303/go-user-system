package handler

import (
	"go-user-system/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) PingHandler(c *gin.Context) {
	response.Success(c, gin.H{
		"message": "success",
	})
}

func (h *HealthHandler) LivezHandler(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "alive",
	})
}

func (h *HealthHandler) ReadyzHandler(c *gin.Context) {
	if h.db == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not initialized")
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not ready")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeReadinessFailed, "database is not ready")
		return
	}

	response.Success(c, gin.H{
		"status": "ready",
	})
}
