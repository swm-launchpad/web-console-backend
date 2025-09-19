package main

import (
	"log"
)

func main() {
	// Initialize application with dependency injection
	app, err := InitializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}
