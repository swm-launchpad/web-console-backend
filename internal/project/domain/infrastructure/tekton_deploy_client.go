// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external services and deployment platforms.
package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// TektonClient defines the interface for interacting with Tekton deployment API.
// This interface abstracts the Tekton EventListener endpoint that triggers PipelineRuns
// for deploying Knative services based on the provided configuration.
//
// The implementation uses HTTP POST requests to communicate with the Tekton EventListener,
// which validates the deployment configuration and creates a PipelineRun resource
// in the Kubernetes cluster.
//
// Authentication is performed using Basic Authentication with credentials configured
// in environment variables.
type TektonClient interface {
	// TriggerDeploy initiates a deployment by sending a deployment request to Tekton API.
	// This triggers the creation of a PipelineRun that will execute the deployment pipeline.
	//
	// The deployment is asynchronous - this method returns immediately after the Tekton
	// EventListener accepts the request. The actual deployment status must be monitored
	// separately using KubeClient.GetPipelineRunStatus().
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - request: Complete deployment configuration including project, containers, volumes, etc.
	//
	// Returns:
	//   - *dto.TektonDeployResponse: Response from Tekton EventListener containing event metadata
	//   - error: An error if the request fails, authentication fails, or Tekton rejects the request
	//
	// The Tekton API returns HTTP 202 Accepted if the deployment request is valid and accepted.
	// The response includes metadata about the triggered event, but does NOT include the
	// PipelineRun name (due to Kubernetes generateName). The PipelineRun must be located
	// using KubeClient.ListPipelineRuns() with the project-id label.
	//
	// Error cases:
	//   - ErrTektonUnavailable: Tekton API is unreachable or misconfigured
	//   - ErrTektonDeploymentFailed: Tekton rejected the deployment (e.g., validation failed)
	//   - ErrInvalidTektonResponse: Response from Tekton could not be parsed
	//
	// Example usage:
	//   request := &dto.TektonDeployRequest{
	//       DeploymentConfigJSON: config,
	//       DryRun: "false",
	//   }
	//   response, err := client.TriggerDeploy(ctx, request)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Deployment triggered: eventID=%s", response.EventID)
	TriggerDeploy(ctx context.Context, request *dto.TektonDeployRequest) (*dto.TektonDeployResponse, error)
}
