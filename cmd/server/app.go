package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type App struct {
	Config         *config.Config
	Database       *sql.DB
	Router         *Router
	OAuthStateRepo repository.OAuthStateRepository
	server         *http.Server
	stopCleanup    chan struct{}
}

func NewApp(cfg *config.Config, database *sql.DB, r *Router, oauthStateRepo repository.OAuthStateRepository) *App {
	return &App{
		Config:         cfg,
		Database:       database,
		Router:         r,
		OAuthStateRepo: oauthStateRepo,
	}
}

func (a *App) Start() error {
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
		log.Printf("Starting server on port %s", a.Config.Server.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	a.waitForShutdown()

	return nil
}

func (a *App) startStateCleanup() {
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
				log.Println("OAuth state cleanup goroutine stopped")
				return
			}
		}
	}()

	log.Println("OAuth state cleanup goroutine started (runs every 1 hour)")
}

func (a *App) cleanupExpiredStates() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := a.OAuthStateRepo.DeleteExpired(ctx)
	if err != nil {
		log.Printf("Failed to cleanup expired OAuth states: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Cleaned up %d expired OAuth states", count)
	}
}

func (a *App) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop cleanup goroutine
	if a.stopCleanup != nil {
		close(a.stopCleanup)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := a.Database.Close(); err != nil {
		log.Printf("Failed to close database connection: %v", err)
	}

	log.Println("Server exited")
}
