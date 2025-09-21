package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
)

func CORS(cfg *config.CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// If no origins specified, allow all origins (for development)
	if len(cfg.AllowedOrigins) == 0 {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
	}

	return cors.New(corsConfig)
}
