// Package service contains domain services that implement business logic
// spanning multiple aggregates or requiring external infrastructure.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	//   log.Printf("Deployment initiated: ID=%d, Status=%s", deployment.DeploymentID, deployment.Status)
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
	//   log.Printf("Refreshed status: %s", deployment.Status)
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
		msg := fmt.Sprintf("Failed to get container config: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 3: Gather volume information
	volumes, err := s.volumeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get volumes: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 4: Build Tekton deployment request
	tektonRequest, err := s.buildTektonRequest(proj, containerConfig, volumes)
	if err != nil {
		msg := fmt.Sprintf("Failed to build Tekton request: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, err
	}

	// Step 5: Trigger deployment via Tekton API
	tektonResponse, err := s.tektonClient.TriggerDeploy(ctx, tektonRequest)
	if err != nil {
		msg := fmt.Sprintf("Failed to trigger Tekton deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrTektonDeploymentFailed
	}

	// Step 6: Update deployment with Tekton event ID
	eventID := tektonResponse.EventID
	if err := d.InitTektonInfo(&eventID, nil); err != nil {
		msg := fmt.Sprintf("Failed to init Tekton info: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return nil, err
	}

	if err := s.deploymentRepo.Save(ctx, d); err != nil {
		msg := fmt.Sprintf("Failed to save deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 7: Start background monitoring in goroutine
	go s.monitorDeployment(context.Background(), projectID, d.DeploymentID)

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

	pipelineRunName, hasRunName := d.TektonPipelineRunName()

	// If no pipeline run name yet, try to find it via label-based lookup
	// This enables "force refresh" to accelerate tracking
	if !hasRunName {
		// Need TektonEventID to perform lookup
		eventID, hasEventID := d.TektonEventID()
		if !hasEventID {
			// CRITICAL DATA INCONSISTENCY: Same issue as in monitorDeployment
			// Deployment is not completed but has no EventID - monitoring impossible
			// Project is stuck in 'deploying' state with no recovery path
			fmt.Printf("========================================\n")
			fmt.Printf("CRITICAL DATA INCONSISTENCY DETECTED (RefreshDeploymentStatus)\n")
			fmt.Printf("========================================\n")
			fmt.Printf("Deployment ID: %d\n", deploymentID)
			fmt.Printf("Project ID: %d\n", d.ProjectID())
			fmt.Printf("Problem: Deployment is not completed (status=%s) but has NO Tekton EventID\n", d.Status())
			fmt.Printf("Impact: Cannot refresh status - project stuck in 'deploying' state\n")
			fmt.Printf("Action: Emergency rollback - setting project->nothing, deployment->backend_tracking_failed\n")
			fmt.Printf("========================================\n")

			msg := "Deployment refresh impossible: no Tekton EventID found (critical data inconsistency)"
			s.handleDeployFailure(ctx, uint(d.ProjectID()), uint(deploymentID), deployment.DeploymentStatusBackendTrackingFailed, &msg)
			return nil, fmt.Errorf("deployment has no Tekton EventID - emergency rollback performed")
		}

		// Find pipeline run by event ID directly
		var err error
		pipelineRunName, err = s.kubeClient.FindPipelineRunNameByEventID(ctx, eventID)
		if err != nil {
			// PipelineRun not created yet by Tekton or other error
			createdAt := d.CreatedAt()
			if time.Since(createdAt) > 5*time.Minute {
				// If more than 5 minutes have passed since deployment creation, mark as tracking failed
				msg := fmt.Sprintf("PipelineRun not found for EventID %s after 5 minutes", eventID)
				s.handleDeployFailure(ctx, uint(d.ProjectID()), uint(deploymentID), deployment.DeploymentStatusBackendTrackingFailed, &msg)
			}
			return nil, fmt.Errorf("pipeline run not found for event %s: %w", eventID, err)
		}

		// Try to update deployment with pipeline run name and running status
		// NOTE: This might fail if background monitoring already marked it as running (race condition)
		runName := pipelineRunName
		if err := d.InitTektonInfo(nil, &runName); err != nil {
			return nil, fmt.Errorf("failed to init Tekton info: %w", err)
		}
	}

	// Query Kubernetes for current status
	status, err := s.kubeClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// If not found, mark as tracking failed(pipeline run deleted)
		if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
			msg := fmt.Sprintf("PipelineRun %s not found in Kubernetes", pipelineRunName)
			s.handleDeployFailure(ctx, uint(d.ProjectID()), uint(deploymentID), deployment.DeploymentStatusBackendTrackingFailed, &msg)
			return nil, err
		}

		// Other errors (connection/authentication) are retriable
		msg := fmt.Sprintf("Failed to get PipelineRun status: %v", err)
		if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingLost, &msg); err != nil {
			return nil, fmt.Errorf("failed to update deployment status: %w", err)
		}
		if err := s.deploymentRepo.Save(ctx, d); err != nil {
			return nil, fmt.Errorf("failed to save deployment: %w", err)
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
	summary *string,
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

		if err := d.UpdateBackendStatus(status, summary); err != nil {
			return fmt.Errorf("failed to update backend status: %w", err)
		}

		if err := s.deploymentRepo.Save(txCtx, d); err != nil {
			return fmt.Errorf("failed to save deployment: %w", err)
		}

		// Reset project operation status to 'nothing' - WITH ROW LOCK
		// Use FindByIDForUpdate to prevent race condition with new deployments
		proj, err := s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			return fmt.Errorf("failed to find project: %w", err)
		}

		// Only reset if still in deploying state
		// This prevents wiping out a new deployment that already started
		if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
			if err := proj.CompleteOperation(); err != nil {
				return fmt.Errorf("failed to complete operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}
		} else {
			// Status already changed by another operation - skip cleanup
			fmt.Printf("INFO: Project %d status is %s (not deploying), skipping cleanup for deployment %d\n",
				projectID, proj.OperationStatus(), deploymentID)
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
	status *dto.PipelineRun,
) error {
	// Determine deployment state based on condition status
	// Status == "Unknown" means PipelineRun is still running or pending
	// Status == "True" means PipelineRun succeeded
	// Status == "False" means PipelineRun failed or was cancelled

	if status.Status == "Unknown" {
		// PipelineRun is still running or pending
		if d.Status() != deployment.DeploymentStatusRunning {
			// Initialize Tekton PipelineRun name if not set
			if _, exists := d.TektonPipelineRunName(); !exists && status.Name != "" {
				name := status.Name
				if err := d.InitTektonInfo(nil, &name); err != nil {
					return err
				}
			}

			// Update to running status
			var summaryPtr *string
			if status.Message != "" {
				summaryPtr = &status.Message
			}
			if err := d.UpdateRunningStatus(summaryPtr, status.StartTime); err != nil {
				return err
			}
		}
		// Save deployment only (no transaction needed for non-terminal states)
		return s.deploymentRepo.Save(ctx, d)
	}

	if status.Status == "True" {
		// PipelineRun succeeded
		// Mark as success and reset project status atomically
		return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if !d.IsCompleted() {
				var summaryPtr *string
				if status.Message != "" {
					summaryPtr = &status.Message
				}
				finishedAt := time.Now()
				if status.CompletionTime != nil {
					finishedAt = *status.CompletionTime
				}
				if err := d.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, summaryPtr, finishedAt); err != nil {
					return fmt.Errorf("failed to mark deployment as complete: %w", err)
				}
			}

			if err := s.deploymentRepo.Save(txCtx, d); err != nil {
				return fmt.Errorf("failed to save deployment: %w", err)
			}

			// Reset project status - WITH ROW LOCK
			proj, err := s.projectRepo.FindByIDForUpdate(txCtx, uint(d.ProjectID()))
			if err != nil {
				return fmt.Errorf("failed to find project: %w", err)
			}

			// Only reset if still in deploying state
			if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
				if err := proj.CompleteOperation(); err != nil {
					return fmt.Errorf("failed to complete operation: %w", err)
				}

				if err := s.projectRepo.Save(txCtx, proj); err != nil {
					return fmt.Errorf("failed to save project: %w", err)
				}
			} else {
				fmt.Printf("INFO: Project %d status is %s (not deploying), skipping reset for deployment %d\n",
					d.ProjectID(), proj.OperationStatus(), d.DeploymentID)
			}

			return nil
		})
	}

	if status.Status == "False" {
		// PipelineRun failed or was cancelled
		// Check reason/message to distinguish between failure and cancellation
		isCancelled := strings.Contains(strings.ToLower(status.Reason), "cancel") ||
			strings.Contains(strings.ToLower(status.Message), "cancel")

		var deploymentStatus deployment.DeploymentStatus
		if isCancelled {
			deploymentStatus = deployment.DeploymentStatusCancelled
		} else {
			deploymentStatus = deployment.DeploymentStatusFailed
		}

		// Mark as failed/cancelled and reset project status atomically
		return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if !d.IsCompleted() {
				var summaryPtr *string
				if status.Message != "" {
					summaryPtr = &status.Message
				}
				finishedAt := time.Now()
				if status.CompletionTime != nil {
					finishedAt = *status.CompletionTime
				}
				if err := d.UpdateCompleteStatus(deploymentStatus, summaryPtr, finishedAt); err != nil {
					return fmt.Errorf("failed to mark deployment as %s: %w", deploymentStatus, err)
				}
			}

			if err := s.deploymentRepo.Save(txCtx, d); err != nil {
				return fmt.Errorf("failed to save deployment: %w", err)
			}

			// Reset project status - WITH ROW LOCK
			proj, err := s.projectRepo.FindByIDForUpdate(txCtx, uint(d.ProjectID()))
			if err != nil {
				return fmt.Errorf("failed to find project: %w", err)
			}

			// Only reset if still in deploying state
			if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
				if err := proj.CompleteOperation(); err != nil {
					return fmt.Errorf("failed to complete operation: %w", err)
				}

				if err := s.projectRepo.Save(txCtx, proj); err != nil {
					return fmt.Errorf("failed to save project: %w", err)
				}
			} else {
				fmt.Printf("INFO: Project %d status is %s (not deploying), skipping reset for deployment %d\n",
					d.ProjectID(), proj.OperationStatus(), d.DeploymentID)
			}

			return nil
		})
	}

	// For unknown statuses, just save deployment
	return s.deploymentRepo.Save(ctx, d)
}

// monitorDeployment monitors a deployment in the background
func (s *deployService) monitorDeployment(ctx context.Context, projectID uint, deploymentID uint) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC in monitorDeployment: %v\n", r)
		}
	}()

	// Polling interval
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d, _ := s.RefreshDeploymentStatus(ctx, uint64(deploymentID))

			if d != nil && d.IsCompleted() {
				// Deployment completed - exit monitoring
				return
			}
		}
	}
}
