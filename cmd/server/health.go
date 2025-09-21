package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(database *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: database,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	// Check database connection
	if err := h.db.Ping(); err != nil {
		response.Error(c, err, nil, response.WithDetails(map[string]interface{}{
			"status":   "unhealthy",
			"database": "disconnected",
		}))
		return
	}

	response.OK(c, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}

func (h *HealthHandler) Root(c *gin.Context) {
	response.OK(c, gin.H{
		"message": "Web Console API",
		"version": "1.0.0",
	})
}
