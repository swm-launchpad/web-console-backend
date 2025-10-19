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
func (s *deployService) DeployProject(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
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

		// Create deployment record with status 'untracked' FIRST
		// We need the deployment ID before setting it on the project
		d = deployment.NewDeployment(projectID)

		if err := s.deploymentRepo.Create(txCtx, d); err != nil {
			return err
		}

		// Change project status to deploying and record which deployment owns it
		if err := proj.StartDeploy(d.DeploymentID); err != nil {
			return err
		}

		if err := s.projectRepo.Save(txCtx, proj); err != nil {
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

// refreshDeploymentStatus queries Kubernetes for the latest status and updates the database
func (s *deployService) refreshDeploymentStatus(ctx context.Context, deploymentID uint64) (*deployment.Deployment, error) {
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
			// Check if the error is a "not found" error or a transient connectivity/auth issue
			if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
				// PipelineRun truly does not exist - only mark as terminal failure after 5 minute grace period
				// This allows time for Tekton to create the PipelineRun (async operation)
				createdAt := d.CreatedAt()
				if time.Since(createdAt) > 5*time.Minute {
					// If more than 5 minutes have passed, mark as tracking failed (terminal)
					msg := fmt.Sprintf("PipelineRun not found for EventID %s after 5 minutes", eventID)
					s.handleDeployFailure(ctx, uint(d.ProjectID()), uint(deploymentID), deployment.DeploymentStatusBackendTrackingFailed, &msg)
				}
				return nil, fmt.Errorf("pipeline run not found for event %s: %w", eventID, err)
			}

			// Other errors (connection/authentication issues) are transient and retriable
			// Update deployment to tracking_lost status (not terminal) and allow retry
			msg := fmt.Sprintf("Failed to find PipelineRun by EventID %s: %v", eventID, err)
			if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingLost, &msg); err != nil {
				return nil, fmt.Errorf("failed to update deployment status: %w", err)
			}
			if err := s.deploymentRepo.Save(ctx, d); err != nil {
				return nil, fmt.Errorf("failed to save deployment: %w", err)
			}
			return nil, err
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

	// ConfigMaps are managed at project level (not yet implemented)
	allConfigMaps := []dto.ConfigMapInfo{}

	// Volumes - only PVC volumes from VolumeRepository (managed at project level)
	allVolumes := s.convertVolumesToDTO(volumes)

	// Create volume_id -> volume_slug mapping for container mount resolution
	volumeMap := make(map[uint]string)
	for _, vol := range volumes {
		volumeMap[vol.VolumeID()] = vol.Slug().String()
	}

	// Convert ContainerInfo to TektonContainerInfo, mapping volume_id to volume_slug
	tektonContainers, err := s.convertContainersToTektonFormat(containerConfig.Containers, volumeMap)
	if err != nil {
		return nil, err
	}

	// Build deployment config
	deploymentConfig := dto.DeploymentConfig{
		ProjectID:    projectIDStr,
		ServiceName:  proj.Slug().String(), // Use project slug for per-project resource isolation in Kubernetes
		Namespace:    s.deployNamespace,
		StableWindow: 180, // constant: 180 seconds
		ConfigMaps:   allConfigMaps,
		Volumes:      allVolumes,
		Containers:   tektonContainers,
	}

	return &dto.TektonDeployRequest{
		DeploymentConfigJSON: deploymentConfig,
		DryRun:               "false",
	}, nil
}

// convertContainersToTektonFormat converts ContainerInfo to TektonContainerInfo,
// mapping volume_id to volume_slug using the provided volumeMap
func (s *deployService) convertContainersToTektonFormat(
	containers []dto.ContainerInfo,
	volumeMap map[uint]string,
) ([]dto.TektonContainerInfo, error) {
	result := make([]dto.TektonContainerInfo, 0, len(containers))

	for _, container := range containers {
		// Convert volume mounts: map volume_id to volume_slug
		tektonMounts := make([]dto.TektonVolumeMount, 0, len(container.VolumeMounts))
		for _, mount := range container.VolumeMounts {
			volumeSlug, exists := volumeMap[mount.VolumeID]
			if !exists {
				// Volume referenced in mount not found - this indicates data inconsistency
				// This should not happen due to foreign key constraints, but if it does (e.g., race condition),
				// we must fail the deployment rather than silently skip the mount
				return nil, fmt.Errorf("%w: container '%s' references volume_id %d which does not exist",
					projecterrors.ErrVolumeMountReferenceNotFound, container.Name, mount.VolumeID)
			}

			tektonMounts = append(tektonMounts, dto.TektonVolumeMount{
				VolumeName: volumeSlug,
				MountPaths: []string{mount.MountPath},
			})
		}

		// Build TektonContainerInfo with all fields from ContainerInfo
		tektonContainer := dto.TektonContainerInfo{
			Name:            container.Name,
			Domain:          container.Domain,
			HealthCheckType: container.HealthCheckType,
			HealthEndpoint:  container.HealthEndpoint,
			Port:            container.Port,
			HealthPort:      container.HealthPort,
			ImageName:       container.ImageName,
			ImageTag:        container.ImageTag,
			EnvVars:         container.EnvVars,
			Secrets:         container.Secrets,
			CPULimit:        container.CPULimit,
			MemoryRequest:   container.MemoryRequest,
			MemoryLimit:     container.MemoryLimit,
			VolumeMounts:    tektonMounts,
		}

		result = append(result, tektonContainer)
	}

	return result, nil
}

// convertVolumesToDTO converts domain volumes to DTO format
// Note: volumes array is now PVC-only, type field is no longer used
func (s *deployService) convertVolumesToDTO(volumes []*volumemodel.Volume) []dto.VolumeInfo {
	result := make([]dto.VolumeInfo, 0, len(volumes))
	for _, v := range volumes {
		// Convert capacity from Mi to string format (e.g., "1024Mi")
		capacityStr := fmt.Sprintf("%dMi", v.Capacity())
		result = append(result, dto.VolumeInfo{
			Name:     v.Slug().String(),
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

		// Only reset if THIS deployment owns the project lock
		// This prevents wiping out a new deployment B that already started after deployment A failed
		if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
			if err := proj.CompleteDeploy(deploymentID); err != nil {
				// CompleteDeploy returns ErrInvalidStatusTransition if deployment doesn't own the lock
				// This is expected in race conditions, so we log and continue
				if errors.Is(err, projecterrors.ErrInvalidStatusTransition) {
					activeDeploymentID, hasActive := proj.ActiveDeploymentID()
					fmt.Printf("INFO: Project %d owned by different deployment (status=%s, active_deployment=%v), skipping cleanup for deployment %d\n",
						projectID, proj.OperationStatus(), activeDeploymentID, deploymentID)
					if hasActive && activeDeploymentID != deploymentID {
						fmt.Printf("WARNING: Race condition detected - deployment %d tried to cleanup but deployment %d owns the project\n",
							deploymentID, activeDeploymentID)
					}
				} else {
					return fmt.Errorf("failed to complete operation: %w", err)
				}
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}
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
		}

		// Update to running status
		var summaryPtr *string
		if status.Message != "" {
			summaryPtr = &status.Message
		}
		if err := d.UpdateRunningStatus(summaryPtr, status.StartTime); err != nil {
			return err
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

			// Only reset if THIS deployment owns the project lock
			if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
				if err := proj.CompleteDeploy(d.DeploymentID); err != nil {
					// CompleteDeploy returns ErrInvalidStatusTransition if deployment doesn't own the lock
					// This is expected in race conditions, so we log and continue
					if errors.Is(err, projecterrors.ErrInvalidStatusTransition) {
						activeDeploymentID, hasActive := proj.ActiveDeploymentID()
						fmt.Printf("INFO: Project %d owned by different deployment (status=%s, active_deployment=%v), skipping reset for deployment %d\n",
							d.ProjectID(), proj.OperationStatus(), activeDeploymentID, d.DeploymentID)
						if hasActive && activeDeploymentID != d.DeploymentID {
							fmt.Printf("WARNING: Race condition avoided - deployment %d tried to reset but deployment %d owns the project\n",
								d.DeploymentID, activeDeploymentID)
						}
					} else {
						return fmt.Errorf("failed to complete operation: %w", err)
					}
				}

				if err := s.projectRepo.Save(txCtx, proj); err != nil {
					return fmt.Errorf("failed to save project: %w", err)
				}
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

			// Only reset if THIS deployment owns the project lock
			if proj.OperationStatus() == value.ProjectOperationStatusDeploying {
				if err := proj.CompleteDeploy(d.DeploymentID); err != nil {
					// CompleteDeploy returns ErrInvalidStatusTransition if deployment doesn't own the lock
					// This is expected in race conditions, so we log and continue
					if errors.Is(err, projecterrors.ErrInvalidStatusTransition) {
						activeDeploymentID, hasActive := proj.ActiveDeploymentID()
						fmt.Printf("INFO: Project %d owned by different deployment (status=%s, active_deployment=%v), skipping reset for deployment %d\n",
							d.ProjectID(), proj.OperationStatus(), activeDeploymentID, d.DeploymentID)
						if hasActive && activeDeploymentID != d.DeploymentID {
							fmt.Printf("WARNING: Race condition avoided - deployment %d tried to reset but deployment %d owns the project\n",
								d.DeploymentID, activeDeploymentID)
						}
					} else {
						return fmt.Errorf("failed to complete operation: %w", err)
					}
				}

				if err := s.projectRepo.Save(txCtx, proj); err != nil {
					return fmt.Errorf("failed to save project: %w", err)
				}
			}

			return nil
		})
	}

	// For unknown statuses, just save deployment
	return s.deploymentRepo.Save(ctx, d)
}

// GetDeploymentStatus retrieves the latest deployment status from the database
func (s *deployService) GetDeploymentStatus(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	// Load project to check active_deployment_id
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// If there's an active deployment, return that one
	if activeDeploymentID, hasActive := proj.ActiveDeploymentID(); hasActive {
		return s.deploymentRepo.FindByID(ctx, activeDeploymentID)
	}

	// Otherwise, return the latest deployment (including completed ones)
	// This allows users to see the last deployment status even after it completes
	return s.deploymentRepo.FindLatestByProjectID(ctx, projectID)
}

// RefreshActiveDeployment queries Kubernetes for the active deployment and updates the database
func (s *deployService) RefreshActiveDeployment(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	// Load project to check active_deployment_id
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if there's an active deployment
	activeDeploymentID, hasActive := proj.ActiveDeploymentID()
	if !hasActive {
		// No active deployment - this means project is not being deployed
		// This is not an error state, just means there's nothing to refresh
		return nil, projecterrors.ErrDeploymentNotFound
	}

	// Refresh the active deployment by querying Kubernetes
	return s.refreshDeploymentStatus(ctx, uint64(activeDeploymentID))
}

// monitorDeployment monitors a deployment in the background
func (s *deployService) monitorDeployment(ctx context.Context, projectID uint, deploymentID uint) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC in monitorDeployment: %v\n", r)
		}
	}()

	// Polling interval
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d, _ := s.refreshDeploymentStatus(ctx, uint64(deploymentID))

			if d != nil && d.IsCompleted() {
				// Deployment completed - exit monitoring
				return
			}
		}
	}
}
