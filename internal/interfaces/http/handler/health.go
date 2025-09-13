package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}

func (h *HealthHandler) Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Web Console API",
		"version": "1.0.0",
	})
}
