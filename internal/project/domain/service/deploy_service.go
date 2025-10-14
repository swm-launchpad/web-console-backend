// Package service contains domain services that implement business logic
// spanning multiple aggregates or requiring external infrastructure.
package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// DeployService defines the interface for deploying projects.
// This service orchestrates the deployment process by coordinating between
// the Project, Deployment, Container, and external infrastructure (Tekton/Kubernetes).
//
// The deployment process follows these high-level steps:
//  1. Validate project state (check operation_status is 'nothing')
//  2. Atomically set project operation_status to 'deploying' and create Deployment record
//  3. Gather all deployment configuration (containers, volumes, project metadata)
//  4. Trigger deployment via Tekton API
//  5. Start background monitoring of deployment status
//  6. Update Deployment and Project status based on monitoring results
//
// The service ensures:
//   - Concurrent deployments for the same project are prevented (via project_operation_status)
//   - Deployment state is tracked accurately (7 possible states)
//   - Failed deployments are properly cleaned up (project status reset to 'nothing')
//   - Monitoring handles both fatal and retriable errors appropriately
type DeployService interface {
	// DeployProject initiates a deployment for the specified project.
	// This method performs validation, state management, and triggers the deployment asynchronously.
	//
	// The deployment is initiated asynchronously - this method returns immediately after
	// the Tekton API accepts the deployment request. The actual deployment status is
	// tracked via background monitoring and can be queried using the Deployment repository.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project to deploy
	//   - userID: The ID of the user initiating the deployment
	//
	// Returns:
	//   - *deployment.Deployment: The created Deployment record with status 'untracked' initially
	//   - error: An error if the deployment cannot be initiated
	//
	// Error cases:
	//   - ErrProjectNotFound: Project does not exist
	//   - ErrProjectAlreadyDeploying: Project is already being deployed (operation_status != 'nothing')
	//   - ErrContainerConfigNotFound: Container configuration is missing for the project
	//   - ErrTektonUnavailable: Tekton API is unreachable
	//   - ErrTektonDeploymentFailed: Tekton rejected the deployment request
	//
	// Example usage:
	//   deployment, err := deployService.DeployProject(ctx, 123, 456)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Deployment initiated: ID=%d, Status=%s", deployment.DeploymentID(), deployment.Status())
	//
	// Post-conditions:
	//   - Project.operation_status is set to 'deploying' (on success)
	//   - Deployment record is created with status 'untracked'
	//   - Background monitoring goroutine is started
	//   - On error, project status is reverted to 'nothing' and Deployment is marked as failed
	DeployProject(ctx context.Context, projectID uint, userID uint) (*deployment.Deployment, error)

	// RefreshDeploymentStatus queries Kubernetes directly to get the latest deployment status
	// and updates the database accordingly. This is used for "force refresh" scenarios where
	// the user wants the most up-to-date status immediately, rather than waiting for the
	// periodic background monitoring.
	//
	// This method is useful when:
	//   - User clicks a "Refresh" button in the UI
	//   - Frontend needs immediate status for critical decisions
	//   - Background monitoring has not updated status for some time
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - deploymentID: The unique identifier of the deployment to refresh
	//
	// Returns:
	//   - *deployment.Deployment: The updated Deployment with latest status from Kubernetes
	//   - error: An error if the operation fails
	//
	// Error cases:
	//   - ErrDeploymentNotFound: Deployment does not exist in database
	//   - ErrKubePipelineRunNotFound: PipelineRun no longer exists in Kubernetes
	//   - ErrKubeConnectionFailed: Cannot connect to Kubernetes API
	//   - ErrKubeAuthFailed: Authentication to Kubernetes failed
	//
	// Example usage:
	//   deployment, err := deployService.RefreshDeploymentStatus(ctx, 789)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Refreshed status: %s", deployment.Status())
	RefreshDeploymentStatus(ctx context.Context, deploymentID uint64) (*deployment.Deployment, error)
}
