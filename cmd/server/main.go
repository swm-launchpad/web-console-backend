package main

import (
	"log"

	"github.com/swm-launchpad/web-console-backend/internal/di"
)

func main() {
	// Initialize application with dependency injection
	app, err := di.InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	
	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}
