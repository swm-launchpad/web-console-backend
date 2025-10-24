package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type App struct {
	Config         *config.Config
	Database       *sql.DB
	Router         *Router
	OAuthStateRepo repository.OAuthStateRepository
	Logger         logger.Logger
	server         *http.Server
	stopCleanup    chan struct{}
}

func NewApp(cfg *config.Config, database *sql.DB, r *Router, oauthStateRepo repository.OAuthStateRepository, log logger.Logger) *App {
	return &App{
		Config:         cfg,
		Database:       database,
		Router:         r,
		OAuthStateRepo: oauthStateRepo,
		Logger:         log,
	}
}

func (a *App) Start() error {
	ctx := context.Background()

	// Setup routes
	a.Router.Setup()

	// Create HTTP server with timeouts for security
	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%s", a.Config.Server.Port),
		Handler:      a.Router.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start OAuth state cleanup goroutine
	a.startStateCleanup()

	// Start server in a goroutine
	go func() {
		a.Logger.Info(ctx, "starting server",
			zap.String("port", a.Config.Server.Port),
			zap.String("gin_mode", a.Config.Server.GinMode),
		)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.Fatal(ctx, "failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	a.waitForShutdown()

	return nil
}

func (a *App) startStateCleanup() {
	ctx := context.Background()
	a.stopCleanup = make(chan struct{})
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		defer ticker.Stop()

		// Run cleanup immediately on startup
		a.cleanupExpiredStates()

		// Then run periodically
		for {
			select {
			case <-ticker.C:
				a.cleanupExpiredStates()
			case <-a.stopCleanup:
				a.Logger.Info(ctx, "oauth state cleanup goroutine stopped")
				return
			}
		}
	}()

	a.Logger.Info(ctx, "oauth state cleanup goroutine started", zap.Duration("interval", 1*time.Hour))
}

func (a *App) cleanupExpiredStates() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := a.OAuthStateRepo.DeleteExpired(ctx)
	if err != nil {
		a.Logger.Error(ctx, "failed to cleanup expired oauth states", zap.Error(err))
		return
	}

	if count > 0 {
		a.Logger.Info(ctx, "cleaned up expired oauth states", zap.Int64("count", count))
	}
}

func (a *App) waitForShutdown() {
	ctx := context.Background()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Logger.Info(ctx, "shutting down server")

	// Stop cleanup goroutine
	if a.stopCleanup != nil {
		close(a.stopCleanup)
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.Logger.Error(ctx, "server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	if err := a.Database.Close(); err != nil {
		a.Logger.Error(ctx, "failed to close database connection", zap.Error(err))
	}

	// Sync logger before exit
	_ = a.Logger.Sync() // Ignore sync errors on stdout/stderr

	a.Logger.Info(ctx, "server exited")
}
