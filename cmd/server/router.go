package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	containerHTTP "github.com/swm-launchpad/web-console-backend/internal/container/handler"
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
	deploymentHandler    *projectHTTP.DeploymentHandler
	containerHandler     *containerHTTP.ContainerHandler
	templateHandler      *containerHTTP.TemplateHandler
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
	deploymentHandler *projectHTTP.DeploymentHandler,
	containerHandler *containerHTTP.ContainerHandler,
	templateHandler *containerHTTP.TemplateHandler,
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
		deploymentHandler:    deploymentHandler,
		containerHandler:     containerHandler,
		templateHandler:      templateHandler,
		authMiddleware:       authMiddleware,
	}
}

func (r *Router) Setup() {
	// Initialize handlers
	healthHandler := NewHealthHandler(r.db)

	// Create rate limiter for email endpoints (3 requests per minute, burst of 5)
	emailRateLimiter := middleware.NewEmailRateLimiter()

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

			// Email verification
			auth.GET("/verify-email", r.verificationHandler.VerifyEmail)
			auth.POST("/resend-verification", emailRateLimiter.RateLimit(), r.verificationHandler.ResendVerificationEmail)

			// Password reset
			auth.POST("/request-password-reset", emailRateLimiter.RateLimit(), r.passwordResetHandler.RequestPasswordReset)
			auth.POST("/reset-password", r.passwordResetHandler.ResetPassword)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(r.authMiddleware.RequireAuth())
		{
			users.GET("/me", r.userHandler.GetCurrentUser)
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

			// Deployment route
			projects.POST("/:id/deploy", r.deploymentHandler.DeployProject)

			// Container routes under project (RESTful)
			projects.POST("/:id/containers", r.containerHandler.CreateContainer)
			projects.GET("/:id/containers", r.containerHandler.ListContainers)
		}

		// Volume routes (protected)
		volumes := v1.Group("/volumes")
		volumes.Use(r.authMiddleware.RequireAuth())
		{
			volumes.POST("", r.volumeHandler.AddVolume)
			volumes.GET("", r.volumeHandler.GetVolumes)
			volumes.DELETE("/:id", r.volumeHandler.RemoveVolume)
		}

		// Container routes (protected)
		// Note: Container creation and listing are available under /projects/:id/containers
		// This group handles operations on individual containers by container ID
		containers := v1.Group("/containers")
		containers.Use(r.authMiddleware.RequireAuth())
		{
			containers.GET("/:id", r.containerHandler.GetContainer)
			containers.PUT("/:id", r.containerHandler.UpdateContainer)
			containers.DELETE("/:id", r.containerHandler.DeleteContainer)

			// Git settings (uses UpdateContainer handler)
			containers.PUT("/:id/git", r.containerHandler.UpdateContainer)

			// Resource limits (uses UpdateContainer handler)
			containers.PUT("/:id/resources", r.containerHandler.UpdateContainer)

			// Environment variables
			containers.GET("/:id/env-vars", r.containerHandler.ListEnvVars)
			containers.POST("/:id/env-vars", r.containerHandler.AddEnvVar)
			containers.PUT("/:id/env-vars/:key", r.containerHandler.UpdateEnvVar)
			containers.DELETE("/:id/env-vars/:key", r.containerHandler.DeleteEnvVar)

			// Networks
			containers.GET("/:id/networks", r.containerHandler.ListNetworks)
			containers.POST("/:id/networks", r.containerHandler.AddNetwork)
			containers.DELETE("/:id/networks/:port", r.containerHandler.DeleteNetwork)

			// Secrets
			containers.GET("/:id/secrets", r.containerHandler.ListSecrets)
			containers.POST("/:id/secrets", r.containerHandler.AddSecret)
			containers.PUT("/:id/secrets/:key", r.containerHandler.UpdateSecret)
			containers.DELETE("/:id/secrets/:key", r.containerHandler.DeleteSecret)

			// Mounts
			containers.POST("/:id/mounts", r.containerHandler.AddMount)
			containers.DELETE("/:id/mounts/:volume_id", r.containerHandler.DeleteMount)
		}

		// Template routes (public - no auth required)
		templates := v1.Group("/templates")
		{
			templates.GET("", r.templateHandler.GetTemplates)
			templates.GET("/:id", r.templateHandler.GetTemplate)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
