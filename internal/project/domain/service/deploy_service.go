// Package service contains domain services that implement business logic
// spanning multiple aggregates or requiring external infrastructure.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
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

// deployService implements the DeployService interface
type deployService struct {
	txManager          db.TxManager
	projectRepo        repository.ProjectRepository
	deploymentRepo     repository.DeploymentRepository
	volumeRepo         repository.VolumeRepository
	containerClient    infrastructure.ContainerClient
	tektonClient       infrastructure.TektonClient
	kubeClient         infrastructure.KubeClient
	deployNamespace    string
	projectServiceName string
}

// NewDeployService creates a new instance of deployService
func NewDeployService(
	txManager db.TxManager,
	projectRepo repository.ProjectRepository,
	deploymentRepo repository.DeploymentRepository,
	volumeRepo repository.VolumeRepository,
	containerClient infrastructure.ContainerClient,
	tektonClient infrastructure.TektonClient,
	kubeClient infrastructure.KubeClient,
	deployNamespace string,
	projectServiceName string,
) DeployService {
	return &deployService{
		txManager:          txManager,
		projectRepo:        projectRepo,
		deploymentRepo:     deploymentRepo,
		volumeRepo:         volumeRepo,
		containerClient:    containerClient,
		tektonClient:       tektonClient,
		kubeClient:         kubeClient,
		deployNamespace:    deployNamespace,
		projectServiceName: projectServiceName,
	}
}

// DeployProject initiates a deployment for the specified project
func (s *deployService) DeployProject(ctx context.Context, projectID uint, userID uint) (*deployment.Deployment, error) {
	// Step 1: Atomically change project status + create deployment in a transaction
	var d *deployment.Deployment
	var proj *projectmodel.Project
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Load project with FOR UPDATE lock
		var err error
		proj, err = s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			return err
		}

		// Check if project is already deploying
		if proj.OperationStatus() != value.ProjectOperationStatusNothing {
			return projecterrors.ErrProjectAlreadyDeploying
		}

		// Change project status to deploying
		if err := proj.StartDeploying(); err != nil {
			return err
		}

		if err := s.projectRepo.Save(txCtx, proj); err != nil {
			return err
		}

		// Create deployment record with status 'untracked'
		d = deployment.NewDeployment(projectID)

		if err := s.deploymentRepo.Create(txCtx, d); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// From now on, any error should rollback project status and mark deployment as failed

	// Step 2: Gather container configuration
	containerConfig, err := s.containerClient.GetContainerConfig(ctx, projectID)
	if err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to get container config: %v", err))
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 3: Gather volume information
	volumes, err := s.volumeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to get volumes: %v", err))
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 4: Build Tekton deployment request
	tektonRequest, err := s.buildTektonRequest(proj, containerConfig, volumes)
	if err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to build Tekton request: %v", err))
		return nil, err
	}

	// Step 5: Trigger deployment via Tekton API
	tektonResponse, err := s.tektonClient.TriggerDeploy(ctx, tektonRequest)
	if err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to trigger Tekton deployment: %v", err))
		return nil, projecterrors.ErrTektonDeploymentFailed
	}

	// Step 6: Update deployment with Tekton event ID
	if err := d.MarkAsTracking(tektonResponse.EventID); err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to mark deployment as tracking: %v", err))
		return nil, err
	}

	if err := s.deploymentRepo.Save(ctx, d); err != nil {
		s.handleDeployFailure(ctx, projectID, d.DeploymentID(), deployment.DeploymentStatusBackendTriggerFailed,
			fmt.Sprintf("Failed to save deployment: %v", err))
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 7: Start background monitoring in goroutine
	go s.monitorDeployment(context.Background(), projectID, d.DeploymentID(), tektonResponse.EventID)

	// Return the deployment record
	return d, nil
}

// RefreshDeploymentStatus queries Kubernetes for the latest status and updates the database
func (s *deployService) RefreshDeploymentStatus(ctx context.Context, deploymentID uint64) (*deployment.Deployment, error) {
	// Load deployment
	d, err := s.deploymentRepo.FindByID(ctx, uint(deploymentID))
	if err != nil {
		return nil, err
	}

	// If deployment is already completed, no need to refresh
	if d.IsCompleted() {
		return d, nil
	}

	pipelineRunName := d.TektonPipelineRunName()

	// If no pipeline run name yet, try to find it via label-based lookup
	// This enables "force refresh" to accelerate tracking
	if pipelineRunName == "" {
		// Need TektonEventID to perform lookup
		if d.TektonEventID() == "" {
			return nil, fmt.Errorf("deployment has no event ID yet, cannot refresh")
		}

		// List all pipeline runs for this project to find the matching one
		runs, err := s.kubeClient.ListPipelineRuns(ctx, uint(d.ProjectID()))
		if err != nil {
			return nil, fmt.Errorf("failed to list pipeline runs: %w", err)
		}

		// Find the pipeline run with matching event ID
		found := false
		for _, run := range runs {
			if run.EventID == d.TektonEventID() {
				pipelineRunName = run.Name
				found = true
				break
			}
		}

		if !found {
			// PipelineRun not created yet by Tekton
			return nil, fmt.Errorf("pipeline run not found for event %s (may not be created yet)", d.TektonEventID())
		}

		// Try to update deployment with pipeline run name
		// NOTE: This might fail if background monitoring already marked it as running (race condition)
		if err := d.MarkAsRunning(pipelineRunName); err != nil {
			// Race condition: background monitoring already marked it as running
			// Reload deployment to get the updated state
			d, reloadErr := s.deploymentRepo.FindByID(ctx, uint(deploymentID))
			if reloadErr != nil {
				return nil, fmt.Errorf("failed to reload deployment after race: %w", reloadErr)
			}
			pipelineRunName = d.TektonPipelineRunName()

			// If still no pipeline run name after reload, the original error was real
			if pipelineRunName == "" {
				return nil, fmt.Errorf("failed to mark as running: %w", err)
			}
			// Successfully resolved race condition, continue with reloaded deployment
		} else {
			// Successfully marked as running, save it
			if err := s.deploymentRepo.Save(ctx, d); err != nil {
				return nil, fmt.Errorf("failed to save deployment: %w", err)
			}
			// Reload to ensure we have fresh state with updated timestamps
			d, err = s.deploymentRepo.FindByID(ctx, uint(deploymentID))
			if err != nil {
				return nil, fmt.Errorf("failed to reload deployment after save: %w", err)
			}
		}
	}

	// Query Kubernetes for current status
	status, err := s.kubeClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// Handle fatal errors
		if projecterrors.IsFatalKubeError(err) {
			// Use handleDeployFailure to atomically update both deployment and project status
			s.handleDeployFailure(ctx, uint(d.ProjectID()), uint(d.DeploymentID()),
				deployment.DeploymentStatusBackendTrackingLost,
				fmt.Sprintf("Kubernetes error during refresh: %v", err))
		}
		return nil, err
	}

	// Update deployment status based on PipelineRun status
	if err := s.updateDeploymentFromKubeStatus(ctx, d, status); err != nil {
		return nil, err
	}

	return d, nil
}

// buildTektonRequest constructs the Tekton deployment request from project data
func (s *deployService) buildTektonRequest(
	proj *projectmodel.Project,
	containerConfig *dto.ContainerDeploymentConfig,
	volumes []*volumemodel.Volume,
) (*dto.TektonDeployRequest, error) {
	// Convert project ID to string for Tekton API
	projectIDStr := fmt.Sprintf("%d", proj.ProjectID())

	// Merge container config volumes with dynamically created PVC volumes
	allVolumes := make([]dto.VolumeInfo, 0)
	allVolumes = append(allVolumes, containerConfig.Volumes...)
	allVolumes = append(allVolumes, s.convertVolumesToDTO(volumes)...)

	// Build deployment config
	deploymentConfig := dto.DeploymentConfig{
		ProjectID:    projectIDStr,
		ServiceName:  s.projectServiceName,
		Namespace:    s.deployNamespace,
		StableWindow: 180, // constant: 180 seconds
		ConfigMaps:   containerConfig.ConfigMaps,
		Volumes:      allVolumes,
		Containers:   containerConfig.Containers,
	}

	return &dto.TektonDeployRequest{
		DeploymentConfigJSON: deploymentConfig,
		DryRun:               "false",
	}, nil
}

// convertVolumesToDTO converts domain volumes to DTO format
func (s *deployService) convertVolumesToDTO(volumes []*volumemodel.Volume) []dto.VolumeInfo {
	result := make([]dto.VolumeInfo, 0, len(volumes))
	for _, v := range volumes {
		volumeType := "pvc"
		// Convert capacity from Mi to string format (e.g., "1024Mi")
		capacityStr := fmt.Sprintf("%dMi", v.Capacity())
		result = append(result, dto.VolumeInfo{
			Name:     v.Name(),
			Type:     &volumeType,
			Capacity: &capacityStr,
		})
	}
	return result
}

// handleDeployFailure handles deployment failure by resetting project status and marking deployment as failed.
// This operation is performed atomically within a transaction to ensure data consistency.
// Uses a fresh context with timeout to ensure cleanup succeeds even if the caller's context is cancelled.
func (s *deployService) handleDeployFailure(
	ctx context.Context,
	projectID uint,
	deploymentID uint,
	status deployment.DeploymentStatus,
	reason string,
) {
	// Create a fresh context with timeout for cleanup operations
	// This ensures cleanup succeeds even if the caller's context is cancelled/expired
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform all state changes atomically within a transaction
	err := s.txManager.RunInTx(cleanupCtx, func(txCtx context.Context) error {
		// Mark deployment as failed
		d, err := s.deploymentRepo.FindByID(txCtx, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to find deployment: %w", err)
		}

		switch status {
		case deployment.DeploymentStatusBackendTriggerFailed:
			if err := d.MarkAsTriggerFailed(reason); err != nil {
				return fmt.Errorf("failed to mark deployment as trigger failed: %w", err)
			}
		case deployment.DeploymentStatusBackendTrackingFailed:
			if err := d.MarkAsTrackingFailed(reason); err != nil {
				return fmt.Errorf("failed to mark deployment as tracking failed: %w", err)
			}
		case deployment.DeploymentStatusBackendTrackingLost:
			if err := d.MarkAsTrackingLost(reason); err != nil {
				return fmt.Errorf("failed to mark deployment as tracking lost: %w", err)
			}
		}

		if err := s.deploymentRepo.Save(txCtx, d); err != nil {
			return fmt.Errorf("failed to save deployment: %w", err)
		}

		// Reset project operation status to 'nothing'
		proj, err := s.projectRepo.FindByID(txCtx, projectID)
		if err != nil {
			return fmt.Errorf("failed to find project: %w", err)
		}

		if err := proj.CompleteOperation(); err != nil {
			return fmt.Errorf("failed to complete operation: %w", err)
		}

		if err := s.projectRepo.Save(txCtx, proj); err != nil {
			return fmt.Errorf("failed to save project: %w", err)
		}

		return nil
	})

	if err != nil {
		// Log the error but don't propagate it since this is a cleanup/recovery operation
		// and we don't want to fail the caller's flow
		fmt.Printf("ERROR: handleDeployFailure failed: %v\n", err)
	}
}

// updateDeploymentFromKubeStatus updates deployment based on Kubernetes PipelineRun status.
// For terminal states (Succeeded/Failed), this operation is performed atomically within a transaction
// to ensure deployment and project status are updated consistently.
func (s *deployService) updateDeploymentFromKubeStatus(
	ctx context.Context,
	d *deployment.Deployment,
	status *dto.PipelineRunStatus,
) error {
	switch status.Status {
	case "Running", "Pending":
		// Update to running status (no project status change needed)
		if d.Status() != deployment.DeploymentStatusRunning {
			if d.TektonPipelineRunName() == "" && status.Name != "" {
				if err := d.MarkAsRunning(status.Name); err != nil {
					return err
				}
			}
		}
		// Save deployment only (no transaction needed for non-terminal states)
		return s.deploymentRepo.Save(ctx, d)

	case "Succeeded":
		// Mark as success and reset project status atomically
		return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if !d.IsCompleted() {
				if err := d.Complete(status.Message); err != nil {
					return fmt.Errorf("failed to mark deployment as complete: %w", err)
				}
			}

			if err := s.deploymentRepo.Save(txCtx, d); err != nil {
				return fmt.Errorf("failed to save deployment: %w", err)
			}

			// Reset project status
			proj, err := s.projectRepo.FindByID(txCtx, uint(d.ProjectID()))
			if err != nil {
				return fmt.Errorf("failed to find project: %w", err)
			}

			if err := proj.CompleteOperation(); err != nil {
				return fmt.Errorf("failed to complete operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}

			return nil
		})

	case "Failed":
		// Mark as failed and reset project status atomically
		return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if !d.IsCompleted() {
				if err := d.Fail(status.Message); err != nil {
					return fmt.Errorf("failed to mark deployment as failed: %w", err)
				}
			}

			if err := s.deploymentRepo.Save(txCtx, d); err != nil {
				return fmt.Errorf("failed to save deployment: %w", err)
			}

			// Reset project status
			proj, err := s.projectRepo.FindByID(txCtx, uint(d.ProjectID()))
			if err != nil {
				return fmt.Errorf("failed to find project: %w", err)
			}

			if err := proj.CompleteOperation(); err != nil {
				return fmt.Errorf("failed to complete operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}

			return nil
		})

	case "Cancelled", "StoppedRunFinally", "CancelledRunFinally":
		// Mark as cancelled and reset project status atomically
		// All three cancellation types (Cancelled, StoppedRunFinally, CancelledRunFinally) are treated as cancelled
		return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if !d.IsCompleted() {
				if err := d.Cancel(status.Message); err != nil {
					return fmt.Errorf("failed to mark deployment as cancelled: %w", err)
				}
			}

			if err := s.deploymentRepo.Save(txCtx, d); err != nil {
				return fmt.Errorf("failed to save deployment: %w", err)
			}

			// Reset project status
			proj, err := s.projectRepo.FindByID(txCtx, uint(d.ProjectID()))
			if err != nil {
				return fmt.Errorf("failed to find project: %w", err)
			}

			if err := proj.CompleteOperation(); err != nil {
				return fmt.Errorf("failed to complete operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}

			return nil
		})
	}

	// For unknown statuses, just save deployment
	return s.deploymentRepo.Save(ctx, d)
}

// monitorDeployment monitors a deployment in the background
func (s *deployService) monitorDeployment(ctx context.Context, projectID uint, deploymentID uint, eventID string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC in monitorDeployment: %v\n", r)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var pipelineRunName string
	failureCount := 0
	maxFailures := 30 // 5 minutes (30 * 10 seconds)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// If we don't have pipeline run name yet, try to find it
			if pipelineRunName == "" {
				runs, err := s.kubeClient.ListPipelineRuns(ctx, projectID)
				if err != nil {
					failureCount++
					fmt.Printf("WARNING: Failed to list PipelineRuns (attempt %d/%d): %v\n", failureCount, maxFailures, err)

					if failureCount >= maxFailures {
						fmt.Printf("CRITICAL: Failed to find PipelineRun after 5 minutes for deployment %d. Manual verification required.\n", deploymentID)
						s.handleDeployFailure(ctx, projectID, deploymentID, deployment.DeploymentStatusBackendTrackingFailed,
							fmt.Sprintf("Failed to find PipelineRun after 5 minutes: %v", err))
						return
					}
					continue
				}

				// Find the pipeline run with matching event ID
				found := false
				for _, run := range runs {
					if run.EventID == eventID {
						pipelineRunName = run.Name
						found = true
						break
					}
				}

				if !found {
					failureCount++
					fmt.Printf("WARNING: PipelineRun not found for event %s (attempt %d/%d)\n", eventID, failureCount, maxFailures)

					if failureCount >= maxFailures {
						fmt.Printf("CRITICAL: PipelineRun not found after 5 minutes for deployment %d. Manual verification required.\n", deploymentID)
						s.handleDeployFailure(ctx, projectID, deploymentID, deployment.DeploymentStatusBackendTrackingFailed,
							fmt.Sprintf("PipelineRun not found after 5 minutes for event %s", eventID))
						return
					}
					continue
				}

				// Found pipeline run, update deployment
				d, err := s.deploymentRepo.FindByID(ctx, deploymentID)
				if err != nil {
					fmt.Printf("ERROR: Failed to load deployment %d: %v\n", deploymentID, err)
					return
				}

				if err := d.MarkAsRunning(pipelineRunName); err != nil {
					fmt.Printf("ERROR: Failed to mark deployment as running: %v\n", err)
					return
				}

				if err := s.deploymentRepo.Save(ctx, d); err != nil {
					fmt.Printf("ERROR: Failed to save deployment: %v\n", err)
					return
				}

				failureCount = 0
				fmt.Printf("INFO: Found PipelineRun %s for deployment %d\n", pipelineRunName, deploymentID)
			}

			// Query pipeline run status
			status, err := s.kubeClient.GetPipelineRunStatus(ctx, pipelineRunName)
			if err != nil {
				// Handle fatal errors
				if projecterrors.IsFatalKubeError(err) {
					fmt.Printf("CRITICAL: Fatal Kubernetes error for deployment %d: %v\n", deploymentID, err)
					s.handleDeployFailure(ctx, projectID, deploymentID, deployment.DeploymentStatusBackendTrackingLost,
						fmt.Sprintf("Fatal Kubernetes error: %v", err))
					return
				}

				// Log warning for retriable errors and continue
				fmt.Printf("WARNING: Failed to get PipelineRun status: %v\n", err)
				continue
			}

			// Load deployment
			d, err := s.deploymentRepo.FindByID(ctx, deploymentID)
			if err != nil {
				fmt.Printf("ERROR: Failed to load deployment %d: %v\n", deploymentID, err)
				return
			}

			// Warn if overwriting a completed status
			if d.IsCompleted() && (status.Status == "Succeeded" || status.Status == "Failed") {
				fmt.Printf("WARNING: Overwriting %s status with %s for deployment %d\n", d.Status(), status.Status, deploymentID)
			}

			// Update deployment status
			if err := s.updateDeploymentFromKubeStatus(ctx, d, status); err != nil {
				fmt.Printf("ERROR: Failed to update deployment status: %v\n", err)
				continue
			}

			// If deployment is completed, stop monitoring
			if d.IsCompleted() {
				fmt.Printf("INFO: Deployment %d completed with status %s\n", deploymentID, d.Status())
				return
			}
		}
	}
}
