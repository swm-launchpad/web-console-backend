package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
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
	githubHandler        *userHTTP.GitHubHandler
	projectHandler       *projectHTTP.ProjectHandler
	projectLogHandler    *projectHTTP.ProjectLogHandler
	volumeHandler        *projectHTTP.VolumeHandler
	deploymentHandler    *projectHTTP.DeploymentHandler
	projectStatusHandler *projectHTTP.ProjectStatusHandler
	containerHandler     *containerHTTP.ContainerHandler
	templateHandler      *containerHTTP.TemplateHandler
	buildLogHandler      *containerHTTP.BuildLogHandler
	settingsHandler      *settings.SettingsHandler
	authMiddleware       *middleware.AuthMiddleware
	loggingMiddleware    *logger.LoggingMiddleware
}

func NewRouter(
	cfg *config.Config,
	database *sql.DB,
	authHandler *userHTTP.AuthHandler,
	userHandler *userHTTP.UserHandler,
	verificationHandler *userHTTP.VerificationHandler,
	passwordResetHandler *userHTTP.PasswordResetHandler,
	githubHandler *userHTTP.GitHubHandler,
	projectHandler *projectHTTP.ProjectHandler,
	projectLogHandler *projectHTTP.ProjectLogHandler,
	volumeHandler *projectHTTP.VolumeHandler,
	deploymentHandler *projectHTTP.DeploymentHandler,
	projectStatusHandler *projectHTTP.ProjectStatusHandler,
	containerHandler *containerHTTP.ContainerHandler,
	templateHandler *containerHTTP.TemplateHandler,
	buildLogHandler *containerHTTP.BuildLogHandler,
	settingsHandler *settings.SettingsHandler,
	authMiddleware *middleware.AuthMiddleware,
	loggingMiddleware *logger.LoggingMiddleware,
) *Router {
	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	r := gin.New()

	// Use custom recovery middleware with logging
	r.Use(loggingMiddleware.RecoveryHandler())

	// Use custom logging middleware (replaces gin.Logger())
	r.Use(loggingMiddleware.Handler())

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
		githubHandler:        githubHandler,
		projectHandler:       projectHandler,
		projectLogHandler:    projectLogHandler,
		volumeHandler:        volumeHandler,
		deploymentHandler:    deploymentHandler,
		projectStatusHandler: projectStatusHandler,
		containerHandler:     containerHandler,
		templateHandler:      templateHandler,
		buildLogHandler:      buildLogHandler,
		settingsHandler:      settingsHandler,
		authMiddleware:       authMiddleware,
		loggingMiddleware:    loggingMiddleware,
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
	r.engine.HEAD("/health", healthHandler.Health)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Settings routes (public) - must be before auth routes
		settingsGroup := v1.Group("/settings")
		{
			settingsGroup.GET("/public", r.settingsHandler.GetPublicSettings)
		}

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

		// GitHub routes
		github := v1.Group("/github")
		{
			// Installation routes
			installation := github.Group("/installation")
			{
				installation.GET("/start", r.authMiddleware.RequireAuth(), r.githubHandler.StartInstallation)
				installation.GET("/callback", r.githubHandler.InstallationCallback) // Public - GitHub redirects here
			}

			// Protected routes
			github.Use(r.authMiddleware.RequireAuth())
			github.POST("/connect", r.githubHandler.ConnectGitHub)
			github.GET("/installations", r.githubHandler.GetInstallations)
			github.DELETE("/installations/:installation_id", r.githubHandler.DisconnectGitHub)
			github.GET("/installations/:installation_id/repositories", r.githubHandler.ListRepositories)
			github.POST("/token", r.githubHandler.GenerateInstallationToken)
		}

		// Project routes (protected)
		projects := v1.Group("/projects")
		projects.Use(r.authMiddleware.RequireAuth())
		{
			projects.POST("", r.projectHandler.CreateProject)
			projects.GET("", r.projectHandler.ListProjects)
			projects.GET("/:slug", r.projectHandler.GetProject)
			projects.PUT("/:slug", r.projectHandler.UpdateProject)
			projects.DELETE("/:slug", r.projectHandler.DeleteProject)

			// Deployment routes
			projects.POST("/:slug/deploy", r.deploymentHandler.DeployProject)

			// Status routes (integrated build and deployment status)
			projects.GET("/:slug/status", r.projectStatusHandler.GetProjectStatus)
			projects.GET("/:slug/status/refresh", r.projectStatusHandler.RefreshProjectStatus)

			// Container routes under project (RESTful)
			projects.POST("/:slug/containers", r.containerHandler.CreateContainer)
			projects.GET("/:slug/containers", r.containerHandler.ListContainers)

			// Application logs (runtime logs)
			projects.POST("/:slug/application-logs/token", r.projectLogHandler.CreateProjectLogToken)
			projects.GET("/:slug/application-logs/history", r.projectLogHandler.GetProjectLogHistory)
		}

		// Volume routes (protected)
		volumes := v1.Group("/volumes")
		volumes.Use(r.authMiddleware.RequireAuth())
		{
			volumes.POST("", r.volumeHandler.AddVolume)
			volumes.GET("", r.volumeHandler.GetVolumes)
			volumes.DELETE("/:slug", r.volumeHandler.RemoveVolume)
		}

		// Container routes (protected)
		// Container slugs are globally unique, no need for project slug in URL
		containers := v1.Group("/containers")
		containers.Use(r.authMiddleware.RequireAuth())
		{
			containers.GET("/:slug", r.containerHandler.GetContainer)
			containers.PUT("/:slug", r.containerHandler.UpdateContainer)
			containers.DELETE("/:slug", r.containerHandler.DeleteContainer)

			// Git settings (uses UpdateContainer handler)
			containers.PUT("/:slug/git", r.containerHandler.UpdateContainer)

			// Resource limits (uses UpdateContainer handler)
			containers.PUT("/:slug/resources", r.containerHandler.UpdateContainer)

			// Environment variables
			containers.GET("/:slug/env-vars", r.containerHandler.ListEnvVars)
			containers.POST("/:slug/env-vars", r.containerHandler.AddEnvVar)
			containers.PUT("/:slug/env-vars/:key", r.containerHandler.UpdateEnvVar)
			containers.DELETE("/:slug/env-vars/:key", r.containerHandler.DeleteEnvVar)

			// Networks
			containers.GET("/:slug/networks", r.containerHandler.ListNetworks)
			containers.POST("/:slug/networks", r.containerHandler.AddNetwork)
			containers.DELETE("/:slug/networks/:port", r.containerHandler.DeleteNetwork)

			// Secrets
			containers.GET("/:slug/secrets", r.containerHandler.ListSecrets)
			containers.POST("/:slug/secrets", r.containerHandler.AddSecret)
			containers.PUT("/:slug/secrets/:key", r.containerHandler.UpdateSecret)
			containers.DELETE("/:slug/secrets/:key", r.containerHandler.DeleteSecret)

			// Build Variables
			containers.GET("/:slug/build-vars", r.containerHandler.ListBuildVars)
			containers.POST("/:slug/build-vars", r.containerHandler.AddBuildVar)
			containers.PUT("/:slug/build-vars/:key", r.containerHandler.UpdateBuildVar)
			containers.DELETE("/:slug/build-vars/:key", r.containerHandler.DeleteBuildVar)

			// Volumes (container sub-resource)
			containers.GET("/:slug/volumes", r.containerHandler.ListVolumes)
			containers.POST("/:slug/volumes", r.containerHandler.AddVolume)
			containers.DELETE("/:slug/volumes/:volume_id", r.containerHandler.DeleteVolume)

			// Build logs
			containers.POST("/:slug/build-log-token", r.buildLogHandler.CreateBuildLogToken)
			containers.GET("/:slug/build-logs/history", r.buildLogHandler.GetBuildLogHistory)
		}

		// Build log streaming WebSocket endpoint (public with token validation)
		// Placed outside auth middleware to allow token-based authentication via query param
		v1.GET("/containers/:slug/build-logs/stream", r.buildLogHandler.StreamBuildLogs)

		// Application log streaming WebSocket endpoint (public with token validation)
		// Placed outside auth middleware to allow token-based authentication via query param
		v1.GET("/projects/:slug/application-logs/stream", r.projectLogHandler.StreamProjectLogs)

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
