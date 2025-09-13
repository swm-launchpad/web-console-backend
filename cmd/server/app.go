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

	"github.com/swm-launchpad/web-console-backend/internal/shared/config"
)

type App struct {
	Config   *config.Config
	Database *sql.DB
	Router   *Router
	server   *http.Server
}

func NewApp(cfg *config.Config, database *sql.DB, r *Router) *App {
	return &App{
		Config:   cfg,
		Database: database,
		Router:   r,
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

func (a *App) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
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
