package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/users/interfaces/http"
)

type Router struct {
	engine         *gin.Engine
	config         *config.Config
	db             *sql.DB
	authHandler    *userHTTP.AuthHandler
	userHandler    *userHTTP.UserHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	database *sql.DB,
	authHandler *userHTTP.AuthHandler,
	userHandler *userHTTP.UserHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Apply CORS middleware
	r.Use(middleware.CORS(&cfg.CORS))

	return &Router{
		engine:         r,
		config:         cfg,
		db:             database,
		authHandler:    authHandler,
		userHandler:    userHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Setup() {
	// Initialize handlers
	healthHandler := NewHealthHandler(r.db)

	// Setup routes
	r.engine.GET("/", healthHandler.Root)
	r.engine.GET("/health", healthHandler.Health)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(r.authMiddleware.RequireAuth())
		{
			users.GET("/me", r.userHandler.GetCurrentUser)
			users.GET("/:id", r.userHandler.GetUserByID)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
