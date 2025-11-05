// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external services and deployment platforms.
package infrastructure

import (
	"context"
)

// TektonCleanupClient defines the interface for interacting with Tekton cleanup API.
// This interface abstracts the Tekton cleanup endpoint that triggers removal of
// Kubernetes and ECR resources associated with a project.
//
// The implementation uses HTTP POST requests to communicate with the Tekton cleanup endpoint,
// which initiates cleanup operations for all resources (Kubernetes deployments, services,
// ECR repositories, etc.) associated with the specified project ID.
//
// Authentication is performed using Basic Authentication with credentials configured
// in environment variables.
type TektonCleanupClient interface {
	// TriggerCleanup initiates cleanup of all Kubernetes and ECR resources for a project.
	// This triggers the deletion of project-related infrastructure resources.
	//
	// The cleanup is asynchronous - this method returns immediately after the Tekton
	// cleanup API accepts the request. The actual cleanup execution happens in the background
	// and does not require monitoring from this service.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project to clean up
	//   - namespace: Kubernetes namespace (default: "application")
	//
	// Returns:
	//   - error: An error if the request fails, authentication fails, or Tekton rejects the request
	//
	// The cleanup request is fire-and-forget. If the API call succeeds (HTTP 2xx), the method
	// returns nil. The actual cleanup status is not monitored by this service.
	//
	// Error cases:
	//   - ErrTektonUnavailable: Tekton cleanup API is unreachable or misconfigured
	//   - ErrTektonCleanupFailed: Tekton rejected the cleanup request
	//
	// Example usage:
	//   err := client.TriggerCleanup(ctx, "my-project-123", "application")
	//   if err != nil {
	//       log.Printf("Warning: Failed to trigger cleanup: %v", err)
	//       // Continue with project deletion anyway
	//   }
	TriggerCleanup(ctx context.Context, projectID, namespace string) error
}
