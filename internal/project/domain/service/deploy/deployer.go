// Package deploy contains domain services for deployment orchestration.
package deploy

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/build"
)

// Deployer defines the interface for deploying projects.
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
type Deployer interface {
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
	//   deployment, err := deployService.DeployProject(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Deployment initiated: ID=%d, Status=%s", deployment.DeploymentID, deployment.Status)
	//
	// Post-conditions:
	//   - Project.operation_status is set to 'deploying' (on success)
	//   - Deployment record is created with status 'untracked'
	//   - Background monitoring goroutine is started
	//   - On error, project status is reverted to 'nothing' and Deployment is marked as failed
	DeployProject(ctx context.Context, projectID uint) (*deployment.Deployment, error)

	// GetDeploymentStatus retrieves the latest deployment status for a project from the database.
	// This is a lightweight read-only operation that does not interact with Kubernetes.
	//
	// This method simply returns the current state stored in the database without
	// making any external API calls.
	//
	// This method is useful when:
	//   - User wants to check deployment status without triggering a Kubernetes query
	//   - Frontend polls for status updates periodically
	//   - Lightweight status checks are needed for UI rendering
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *deployment.Deployment: The latest deployment record from the database
	//   - error: An error if the operation fails
	//
	// Error cases:
	//   - ErrDeploymentNotFound: No deployment exists for the project
	//   - ErrDatabaseOperation: Database query failed
	//
	// Example usage:
	//   deployment, err := deployService.GetDeploymentStatus(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Current status: %s", deployment.Status)
	GetDeploymentStatus(ctx context.Context, projectID uint) (*deployment.Deployment, error)

	// RefreshActiveDeployment queries Kubernetes for the active deployment of a project
	// and updates the database accordingly. This method uses project.active_deployment_id
	// to identify which deployment to refresh.
	//
	// This method takes a projectID and uses the deployment locking mechanism
	// to find the active deployment.
	//
	// This method is useful when:
	//   - User wants to force refresh the current deployment
	//   - Frontend needs immediate status update via project ID
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *deployment.Deployment: The updated Deployment with latest status from Kubernetes
	//   - error: An error if the operation fails
	//
	// Error cases:
	//   - ErrProjectNotFound: Project does not exist
	//   - ErrDeploymentNotFound: No active deployment for the project
	//   - ErrKubePipelineRunNotFound: PipelineRun no longer exists in Kubernetes
	//   - ErrKubeConnectionFailed: Cannot connect to Kubernetes API
	//
	// Example usage:
	//   deployment, err := deployService.RefreshActiveDeployment(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Refreshed status: %s", deployment.Status)
	RefreshActiveDeployment(ctx context.Context, projectID uint) (*deployment.Deployment, error)

	// RefreshActiveBuildStatuses queries Kubernetes for all active builds of a project
	// and updates the database accordingly. This method handles the complete build refresh
	// workflow including project status reset when all builds complete.
	//
	// This method:
	//   - Fetches all containers for the project
	//   - Refreshes build history for each container from Kubernetes (if not terminal)
	//   - Automatically resets project.operation_status to 'nothing' when all builds complete
	//   - Prevents projects from being stuck in 'building' state
	//
	// This method is useful when:
	//   - User wants to force refresh build statuses
	//   - Background monitoring goroutine crashed and builds need manual refresh
	//   - Frontend needs immediate build status update
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - []*build_history.BuildHistory: List of build histories with latest status from Kubernetes
	//   - bool: true if project status was reset to 'nothing' (all builds completed)
	//   - error: An error if the operation fails
	//
	// Error cases:
	//   - ErrProjectNotFound: Project does not exist
	//   - ErrContainerNotFound: No containers configured for the project
	//   - ErrKubePipelineRunNotFound: PipelineRun no longer exists in Kubernetes
	//   - ErrKubeConnectionFailed: Cannot connect to Kubernetes API
	//
	// Example usage:
	//   buildHistories, projectReset, err := deployService.RefreshActiveBuildStatuses(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Refreshed %d builds, project reset: %t", len(buildHistories), projectReset)
	RefreshActiveBuildStatuses(ctx context.Context, projectID uint) ([]*build_history.BuildHistory, bool, error)

	// BuildAndDeployProject builds all containers for a project, then deploys them.
	// This method orchestrates the complete CI/CD pipeline: build → deploy.
	//
	// The method performs validation and immediately returns with a 202 response,
	// then executes builds and deployment in a background goroutine.
	//
	// Process:
	//  1. Validate project state (check operation_status is 'nothing')
	//  2. Validate containers exist
	//  3. Atomically set project operation_status to 'building'
	//  4. Return immediately (for 202 response)
	//  5. Background goroutine executes:
	//     - Build all containers in parallel (via build.Orchestrator)
	//     - Update container metadata after successful builds
	//     - Deploy project if all builds succeed
	//     - Reset project status to 'nothing' on any error
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - error: Returns error only if validation fails or status cannot be set
	//
	// Error cases:
	//   - ErrProjectNotFound: Project does not exist
	//   - ErrProjectAlreadyDeploying: Project is already being deployed/built (operation_status != 'nothing')
	//   - ErrContainerConfigNotFound: No containers configured for the project
	//
	// Example usage:
	//   err := deployService.BuildAndDeployProject(ctx, 123)
	//   if err != nil {
	//       return err  // Validation failed
	//   }
	//   // Return 202 Accepted - builds are running in background
	//
	// Post-conditions:
	//   - Project.operation_status is set to 'building' (on success)
	//   - Background goroutine is started for build+deploy orchestration
	//   - On error during build/deploy, project status is reverted to 'nothing'
	BuildAndDeployProject(ctx context.Context, projectID uint) error
}

// deployService implements the Deployer interface
type deployService struct {
	txManager          db.TxManager
	projectRepo        repository.ProjectRepository
	deploymentRepo     repository.DeploymentRepository
	buildHistoryRepo   repository.BuildHistoryRepository
	volumeRepo         repository.VolumeRepository
	containerClient    infrastructure.ContainerClient
	tektonClient       infrastructure.TektonClient
	kubeClient         infrastructure.KubeClient
	kubeBuildClient    infrastructure.KubeBuildClient
	buildOrchestrator  build.Orchestrator
	buildPostProcessor build.PostProcessor
	deployNamespace    string
	projectServiceName string
	logger             logger.Logger
}

// NewDeployer creates a new instance of deployService
func NewDeployer(
	txManager db.TxManager,
	projectRepo repository.ProjectRepository,
	deploymentRepo repository.DeploymentRepository,
	buildHistoryRepo repository.BuildHistoryRepository,
	volumeRepo repository.VolumeRepository,
	containerClient infrastructure.ContainerClient,
	tektonClient infrastructure.TektonClient,
	kubeClient infrastructure.KubeClient,
	kubeBuildClient infrastructure.KubeBuildClient,
	buildOrchestrator build.Orchestrator,
	buildPostProcessor build.PostProcessor,
	deployNamespace string,
	projectServiceName string,
	log logger.Logger,
) Deployer {
	return &deployService{
		txManager:          txManager,
		projectRepo:        projectRepo,
		deploymentRepo:     deploymentRepo,
		buildHistoryRepo:   buildHistoryRepo,
		volumeRepo:         volumeRepo,
		containerClient:    containerClient,
		tektonClient:       tektonClient,
		kubeClient:         kubeClient,
		kubeBuildClient:    kubeBuildClient,
		buildOrchestrator:  buildOrchestrator,
		buildPostProcessor: buildPostProcessor,
		deployNamespace:    deployNamespace,
		projectServiceName: projectServiceName,
		logger:             log,
	}
}

// DeployProject initiates a deployment for the specified project
