package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	projectHTTP "github.com/swm-launchpad/web-console-backend/internal/project/handler"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/user/handler"
)

type Router struct {
	engine               *gin.Engine
	config               *config.Config
	db                   *sql.DB
	authHandler          *userHTTP.AuthHandler
	userHandler          *userHTTP.UserHandler
	verificationHandler  *userHTTP.VerificationHandler
	passwordResetHandler *userHTTP.PasswordResetHandler
	projectHandler       *projectHTTP.ProjectHandler
	volumeHandler        *projectHTTP.VolumeHandler
	authMiddleware       *middleware.AuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	database *sql.DB,
	authHandler *userHTTP.AuthHandler,
	userHandler *userHTTP.UserHandler,
	verificationHandler *userHTTP.VerificationHandler,
	passwordResetHandler *userHTTP.PasswordResetHandler,
	projectHandler *projectHTTP.ProjectHandler,
	volumeHandler *projectHTTP.VolumeHandler,
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
		engine:               r,
		config:               cfg,
		db:                   database,
		authHandler:          authHandler,
		userHandler:          userHandler,
		verificationHandler:  verificationHandler,
		passwordResetHandler: passwordResetHandler,
		projectHandler:       projectHandler,
		volumeHandler:        volumeHandler,
		authMiddleware:       authMiddleware,
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
			auth.GET("/verify-email", r.verificationHandler.VerifyEmail)
			auth.POST("/resend-verification", r.verificationHandler.ResendVerificationEmail)
			auth.POST("/request-password-reset", r.passwordResetHandler.RequestPasswordReset)
			auth.POST("/reset-password", r.passwordResetHandler.ResetPassword)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(r.authMiddleware.RequireAuth())
		{
			users.GET("/me", r.userHandler.GetCurrentUser)
			users.PUT("/me", r.userHandler.UpdateProfile)
			users.PUT("/me/password", r.userHandler.ChangePassword)
			users.GET("/:id", r.userHandler.GetUserByID)
		}

		// Project routes (protected)
		projects := v1.Group("/projects")
		projects.Use(r.authMiddleware.RequireAuth())
		{
			projects.POST("", r.projectHandler.CreateProject)
			projects.GET("", r.projectHandler.ListProjects)
			projects.GET("/:id", r.projectHandler.GetProject)
			projects.PUT("/:id", r.projectHandler.UpdateProject)
			projects.DELETE("/:id", r.projectHandler.DeleteProject)
		}

		// Volume routes (protected)
		volumes := v1.Group("/volumes")
		volumes.Use(r.authMiddleware.RequireAuth())
		{
			volumes.POST("", r.volumeHandler.AddVolume)
			volumes.GET("", r.volumeHandler.GetVolumes)
			volumes.DELETE("/:id", r.volumeHandler.RemoveVolume)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
