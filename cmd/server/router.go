package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/shared/config"
	"github.com/swm-launchpad/web-console-backend/internal/shared/middleware"
)

type Router struct {
	engine *gin.Engine
	config *config.Config
	db     *sql.DB
}

func NewRouter(cfg *config.Config, database *sql.DB) *Router {
	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Apply CORS middleware
	r.Use(middleware.CORS(&cfg.CORS))

	return &Router{
		engine: r,
		config: cfg,
		db:     database,
	}
}

func (r *Router) Setup() {
	// Initialize handlers
	healthHandler := NewHealthHandler(r.db)

	// Setup routes
	r.engine.GET("/", healthHandler.Root)
	r.engine.GET("/health", healthHandler.Health)

	// API v1 routes will be added when needed
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
