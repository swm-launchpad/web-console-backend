package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func main() {
	// Initialize application with dependency injection
	app, err := InitializeApp()
	if err != nil {
		// Logger may not be available yet, so write to stderr
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Start the application
	// If start fails, logger is already available via app.Logger
	if err := app.Start(); err != nil {
		app.Logger.Fatal(context.Background(), "failed to start application", zap.Error(err))
	}
}
