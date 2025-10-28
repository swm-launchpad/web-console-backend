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
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"go.uber.org/zap"
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
	//     - Build all containers in parallel (via BuildOrchestrator)
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

// deployService implements the DeployService interface
type deployService struct {
	txManager          db.TxManager
	projectRepo        repository.ProjectRepository
	deploymentRepo     repository.DeploymentRepository
	volumeRepo         repository.VolumeRepository
	containerClient    infrastructure.ContainerClient
	tektonClient       infrastructure.TektonClient
	kubeClient         infrastructure.KubeClient
	buildOrchestrator  BuildOrchestrator
	buildPostProcessor BuildPostProcessor
	deployNamespace    string
	projectServiceName string
	logger             logger.Logger
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
	buildOrchestrator BuildOrchestrator,
	buildPostProcessor BuildPostProcessor,
	deployNamespace string,
	projectServiceName string,
	log logger.Logger,
) DeployService {
	return &deployService{
		txManager:          txManager,
		projectRepo:        projectRepo,
		deploymentRepo:     deploymentRepo,
		volumeRepo:         volumeRepo,
		containerClient:    containerClient,
		tektonClient:       tektonClient,
		kubeClient:         kubeClient,
		buildOrchestrator:  buildOrchestrator,
		buildPostProcessor: buildPostProcessor,
		deployNamespace:    deployNamespace,
		projectServiceName: projectServiceName,
		logger:             log,
	}
}

// DeployProject initiates a deployment for the specified project
func (s *deployService) DeployProject(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	s.logger.Info(ctx, "deploy project started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Atomically change project status + create deployment in a transaction
	var d *deployment.Deployment
	var proj *projectmodel.Project
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Load project with FOR UPDATE lock
		var err error
		proj, err = s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			s.logger.Error(ctx, "failed to find project for update",
				zap.Uint("project_id", projectID),
				zap.Error(err),
			)
			return err
		}

		// Check if project is already deploying
		if proj.OperationStatus() != value.ProjectOperationStatusNothing {
			s.logger.Error(ctx, "project already deploying",
				zap.Uint("project_id", projectID),
				zap.String("operation_status", string(proj.OperationStatus())),
			)
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
		s.logger.Error(ctx, "failed to create deployment in transaction",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "deployment record created",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
	)

	// From now on, any error should rollback project status and mark deployment as failed

	// Step 2: Gather container configuration
	containerConfig, err := s.containerClient.GetContainerConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container config",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get container config: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 3: Gather volume information
	volumes, err := s.volumeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get volumes",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get volumes: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 4: Build Tekton deployment request
	tektonRequest, err := s.buildTektonRequest(proj, containerConfig, volumes)
	if err != nil {
		s.logger.Error(ctx, "failed to build Tekton request",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to build Tekton request: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, err
	}

	// Step 5: Trigger deployment via Tekton API
	tektonResponse, err := s.tektonClient.TriggerDeploy(ctx, tektonRequest)
	if err != nil {
		s.logger.Error(ctx, "failed to trigger Tekton deployment",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to trigger Tekton deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return nil, projecterrors.ErrTektonDeploymentFailed
	}

	s.logger.Info(ctx, "Tekton deployment triggered successfully",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
		zap.String("event_id", tektonResponse.EventID),
	)

	// Step 6: Update deployment with Tekton event ID
	eventID := tektonResponse.EventID
	if err := d.InitTektonInfo(&eventID, nil); err != nil {
		s.logger.Error(ctx, "failed to init Tekton info",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to init Tekton info: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return nil, err
	}

	if err := s.deploymentRepo.Save(ctx, d); err != nil {
		s.logger.Error(ctx, "failed to save deployment",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to save deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Step 7: Start background monitoring in goroutine
	s.logger.Info(ctx, "starting background deployment monitoring",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
	)
	// Create a detached context that preserves request_id and user_id for logging correlation
	// but doesn't inherit cancellation from the original request
	monitorCtx := logger.DetachContext(ctx)
	go s.monitorDeployment(monitorCtx, projectID, d.DeploymentID)

	s.logger.Info(ctx, "deploy project completed",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
	)
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
			s.logger.Error(ctx, "CRITICAL DATA INCONSISTENCY DETECTED (RefreshDeploymentStatus)",
				zap.Uint64("deployment_id", deploymentID),
				zap.Uint("project_id", d.ProjectID()),
				zap.String("status", string(d.Status())),
				zap.String("problem", "Deployment is not completed but has NO Tekton EventID"),
				zap.String("impact", "Cannot refresh status - project stuck in 'deploying' state"),
				zap.String("action", "Emergency rollback - setting project->nothing, deployment->backend_tracking_failed"),
			)

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
					s.logger.Info(ctx, "Project owned by different deployment, skipping cleanup",
						zap.Uint("project_id", projectID),
						zap.String("project_status", string(proj.OperationStatus())),
						zap.Uint("active_deployment_id", activeDeploymentID),
						zap.Uint("cleanup_deployment_id", deploymentID),
					)
					if hasActive && activeDeploymentID != deploymentID {
						s.logger.Warn(ctx, "Race condition detected - deployment tried to cleanup but different deployment owns the project",
							zap.Uint("cleanup_deployment_id", deploymentID),
							zap.Uint("owner_deployment_id", activeDeploymentID),
						)
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
		s.logger.Error(ctx, "handleDeployFailure failed",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", deploymentID),
			zap.Error(err),
		)
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
						s.logger.Info(ctx, "Project owned by different deployment, skipping reset",
							zap.Uint("project_id", d.ProjectID()),
							zap.String("project_status", string(proj.OperationStatus())),
							zap.Uint("active_deployment_id", activeDeploymentID),
							zap.Uint("reset_deployment_id", d.DeploymentID),
						)
						if hasActive && activeDeploymentID != d.DeploymentID {
							s.logger.Warn(ctx, "Race condition avoided - deployment tried to reset but different deployment owns the project",
								zap.Uint("reset_deployment_id", d.DeploymentID),
								zap.Uint("owner_deployment_id", activeDeploymentID),
							)
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
						s.logger.Info(ctx, "Project owned by different deployment, skipping reset",
							zap.Uint("project_id", d.ProjectID()),
							zap.String("project_status", string(proj.OperationStatus())),
							zap.Uint("active_deployment_id", activeDeploymentID),
							zap.Uint("reset_deployment_id", d.DeploymentID),
						)
						if hasActive && activeDeploymentID != d.DeploymentID {
							s.logger.Warn(ctx, "Race condition avoided - deployment tried to reset but different deployment owns the project",
								zap.Uint("reset_deployment_id", d.DeploymentID),
								zap.Uint("owner_deployment_id", activeDeploymentID),
							)
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
	s.logger.Info(ctx, "get deployment status started",
		zap.Uint("project_id", projectID),
	)

	// Load project to check active_deployment_id
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// If there's an active deployment, return that one
	if activeDeploymentID, hasActive := proj.ActiveDeploymentID(); hasActive {
		d, err := s.deploymentRepo.FindByID(ctx, activeDeploymentID)
		if err != nil {
			s.logger.Error(ctx, "failed to find active deployment",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", activeDeploymentID),
				zap.Error(err),
			)
			return nil, err
		}
		s.logger.Info(ctx, "get deployment status completed (active)",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.String("status", string(d.Status())),
		)
		return d, nil
	}

	// Otherwise, return the latest deployment (including completed ones)
	// This allows users to see the last deployment status even after it completes
	d, err := s.deploymentRepo.FindLatestByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find latest deployment",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}
	s.logger.Info(ctx, "get deployment status completed (latest)",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
		zap.String("status", string(d.Status())),
	)
	return d, nil
}

// RefreshActiveDeployment queries Kubernetes for the active deployment and updates the database
func (s *deployService) RefreshActiveDeployment(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	s.logger.Info(ctx, "refresh active deployment started",
		zap.Uint("project_id", projectID),
	)

	// Load project to check active_deployment_id
	proj, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// Check if there's an active deployment
	activeDeploymentID, hasActive := proj.ActiveDeploymentID()
	if !hasActive {
		s.logger.Error(ctx, "no active deployment for project",
			zap.Uint("project_id", projectID),
		)
		// No active deployment - this means project is not being deployed
		// This is not an error state, just means there's nothing to refresh
		return nil, projecterrors.ErrDeploymentNotFound
	}

	// Refresh the active deployment by querying Kubernetes
	d, err := s.refreshDeploymentStatus(ctx, uint64(activeDeploymentID))
	if err != nil {
		s.logger.Error(ctx, "failed to refresh deployment status",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", activeDeploymentID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "refresh active deployment completed",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
		zap.String("status", string(d.Status())),
	)
	return d, nil
}

// monitorDeployment monitors a deployment in the background
func (s *deployService) monitorDeployment(ctx context.Context, projectID uint, deploymentID uint) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "PANIC in monitorDeployment",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", deploymentID),
				zap.Any("panic", r),
			)
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

// deployProjectInternal deploys a project using the provided container configuration.
// This method performs the deployment and monitors it directly (no background goroutine).
//
// The method is used by BuildAndDeployProject to deploy after builds complete.
// It accepts container configuration as a parameter to ensure consistency with the
// built artifacts - the configuration represents the state when builds were initiated.
//
// Process:
//  1. Atomically set project status to 'deploying' and create Deployment record
//  2. Gather deployment configuration (volumes, project metadata)
//  3. Trigger deployment via Tekton API
//  4. Monitor deployment directly with 10-second polling (30-minute timeout)
//  5. Update deployment and project status when terminal state is reached
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - projectID: The unique identifier of the project
//   - containerConfig: Container configuration snapshot captured before builds
//
// Returns:
//   - error: Returns error if deployment cannot be initiated or monitoring fails
//
// Note: This method does NOT spawn a goroutine - it performs monitoring synchronously.
// This allows the caller (buildAndDeployInBackground) to maintain a single goroutine
// for the entire build+deploy flow.
func (s *deployService) deployProjectInternal(
	ctx context.Context,
	projectID uint,
	containerConfig *dto.ContainerDeploymentConfig,
) error {
	s.logger.Info(ctx, "deployProjectInternal started",
		zap.Uint("project_id", projectID),
		zap.Int("container_count", len(containerConfig.Containers)),
	)

	// Step 1: Atomically change project status to 'deploying' + create deployment in a transaction
	var d *deployment.Deployment
	var proj *projectmodel.Project
	err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Load project with FOR UPDATE lock
		var err error
		proj, err = s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			s.logger.Error(ctx, "failed to find project for update",
				zap.Uint("project_id", projectID),
				zap.Error(err),
			)
			return err
		}

		// Check if project is in 'building' status
		// It should be 'building' since we're called from buildAndDeployInBackground
		if proj.OperationStatus() != value.ProjectOperationStatusBuilding {
			s.logger.Error(ctx, "project not in building status",
				zap.Uint("project_id", projectID),
				zap.String("operation_status", string(proj.OperationStatus())),
			)
			return fmt.Errorf("project not in building status: %s", proj.OperationStatus())
		}

		// Create deployment record with status 'untracked' FIRST
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
		s.logger.Error(ctx, "failed to create deployment in transaction",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "deployment record created, project status set to deploying",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
	)

	// From now on, any error should rollback project status and mark deployment as failed

	// Step 2: Gather volume information
	volumes, err := s.volumeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get volumes",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get volumes: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return projecterrors.ErrDatabaseOperation
	}

	// Step 3: Build Tekton deployment request using provided container configuration
	tektonRequest, err := s.buildTektonRequest(proj, containerConfig, volumes)
	if err != nil {
		s.logger.Error(ctx, "failed to build Tekton request",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to build Tekton request: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return err
	}

	// Step 4: Trigger deployment via Tekton API
	tektonResponse, err := s.tektonClient.TriggerDeploy(ctx, tektonRequest)
	if err != nil {
		s.logger.Error(ctx, "failed to trigger Tekton deployment",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to trigger Tekton deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
		return projecterrors.ErrTektonDeploymentFailed
	}

	s.logger.Info(ctx, "Tekton deployment triggered successfully",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
		zap.String("event_id", tektonResponse.EventID),
	)

	// Step 5: Update deployment with Tekton event ID
	eventID := tektonResponse.EventID
	if err := d.InitTektonInfo(&eventID, nil); err != nil {
		s.logger.Error(ctx, "failed to init Tekton info",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to init Tekton info: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return err
	}

	if err := s.deploymentRepo.Save(ctx, d); err != nil {
		s.logger.Error(ctx, "failed to save deployment",
			zap.Uint("project_id", projectID),
			zap.Uint("deployment_id", d.DeploymentID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to save deployment: %v", err)
		s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
		return projecterrors.ErrDatabaseOperation
	}

	// Step 6: Monitor deployment directly (no goroutine)
	s.logger.Info(ctx, "starting direct deployment monitoring",
		zap.Uint("project_id", projectID),
		zap.Uint("deployment_id", d.DeploymentID),
	)

	// Use ticker for 10-second polling intervals
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Set 30-minute timeout for deployment monitoring
	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - mark deployment as tracking lost
			s.logger.Error(ctx, "deployment monitoring cancelled",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", d.DeploymentID),
				zap.Error(ctx.Err()),
			)
			msg := fmt.Sprintf("Deployment monitoring cancelled: %v", ctx.Err())
			s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingLost, &msg)
			return ctx.Err()

		case <-timeout:
			// Timeout reached - mark deployment as tracking failed
			s.logger.Error(ctx, "deployment monitoring timeout",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", d.DeploymentID),
			)
			msg := "Deployment monitoring timeout after 30 minutes"
			s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTrackingFailed, &msg)
			return fmt.Errorf("deployment monitoring timeout")

		case <-ticker.C:
			// Poll deployment status
			refreshedDeployment, err := s.refreshDeploymentStatus(ctx, uint64(d.DeploymentID))
			if err != nil {
				// Log error but continue monitoring (transient errors are retriable)
				s.logger.Warn(ctx, "failed to refresh deployment status, will retry",
					zap.Uint("project_id", projectID),
					zap.Uint("deployment_id", d.DeploymentID),
					zap.Error(err),
				)
				continue
			}

			// Check if deployment reached terminal state
			if refreshedDeployment != nil && refreshedDeployment.IsCompleted() {
				s.logger.Info(ctx, "deployment reached terminal state",
					zap.Uint("project_id", projectID),
					zap.Uint("deployment_id", d.DeploymentID),
					zap.String("status", string(refreshedDeployment.Status())),
				)
				// Deployment completed - exit monitoring
				// Note: refreshDeploymentStatus already handled project status reset
				return nil
			}
		}
	}
}

// BuildAndDeployProject builds all containers for a project, then deploys them
func (s *deployService) BuildAndDeployProject(ctx context.Context, projectID uint) error {
	s.logger.Info(ctx, "build and deploy project started",
		zap.Uint("project_id", projectID),
	)

	// Phase 1: Validate project exists first (authorization/ownership check must come before resource checks)
	// This ensures we return ErrProjectNotFound for non-existent/unauthorized projects,
	// not ErrContainerConfigNotFound which would leak information about other users' projects
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err // Returns ErrProjectNotFound if project doesn't exist
	}

	// Phase 2: Check container configuration (only after confirming project exists)
	containerConfig, err := s.containerClient.GetContainerBuildConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container build config",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return projecterrors.ErrContainerConfigNotFound
	}

	if len(containerConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers found for project",
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrContainerConfigNotFound
	}

	// Phase 3: Atomically change project status to 'building'
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Reload project with FOR UPDATE lock (need to lock for status update)
		proj, err := s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			s.logger.Error(ctx, "failed to find project for update",
				zap.Uint("project_id", projectID),
				zap.Error(err),
			)
			return err
		}

		// Check if project is already deploying or building
		if proj.OperationStatus() != value.ProjectOperationStatusNothing {
			s.logger.Error(ctx, "project already in operation",
				zap.Uint("project_id", projectID),
				zap.String("operation_status", string(proj.OperationStatus())),
			)
			return projecterrors.ErrProjectAlreadyDeploying
		}

		// Change project status to building
		if err := proj.StartBuild(); err != nil {
			return err
		}

		if err := s.projectRepo.Save(txCtx, proj); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to set project status to building",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "project status set to building",
		zap.Uint("project_id", projectID),
	)

	// Phase 3: Start background goroutine for build and deploy
	// Create a detached context that preserves request_id and user_id for logging correlation
	// but doesn't inherit cancellation from the original request
	backgroundCtx := logger.DetachContext(ctx)
	go s.buildAndDeployInBackground(backgroundCtx, projectID)

	s.logger.Info(ctx, "build and deploy project initiated",
		zap.Uint("project_id", projectID),
	)

	return nil
}

// buildAndDeployInBackground executes build and deploy operations in the background
func (s *deployService) buildAndDeployInBackground(ctx context.Context, projectID uint) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "PANIC in buildAndDeployInBackground",
				zap.Uint("project_id", projectID),
				zap.Any("panic", r),
			)
			// Attempt to reset project status on panic
			msg := fmt.Sprintf("Panic during build and deploy: %v", r)
			s.handleBuildError(ctx, projectID, &msg)
		}
	}()

	s.logger.Info(ctx, "background build and deploy started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Get container build configuration AND deployment configuration
	// We capture both snapshots BEFORE starting builds to ensure consistency.
	// This way, the deployment uses the same configuration state that existed when builds started.
	buildConfig, err := s.containerClient.GetContainerBuildConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container build config in background",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get container build config: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	deploymentConfig, err := s.containerClient.GetContainerConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container deployment config in background",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get container deployment config: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	s.logger.Info(ctx, "retrieved container configurations",
		zap.Uint("project_id", projectID),
		zap.Int("build_container_count", len(buildConfig.Containers)),
		zap.Int("deploy_container_count", len(deploymentConfig.Containers)),
	)

	// Guard: Verify containers still exist (race condition: container deleted between validation and here)
	if len(buildConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers found for build after status flip",
			zap.Uint("project_id", projectID),
			zap.String("reason", "containers were deleted between initial validation and background execution"),
		)
		msg := "No containers found for build (deleted after status flip)"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	// Convert []BuildContainerInfo to []*BuildContainerInfo
	containerPointers := make([]*dto.BuildContainerInfo, len(buildConfig.Containers))
	for i := range buildConfig.Containers {
		containerPointers[i] = &buildConfig.Containers[i]
	}

	// Step 2: Execute builds in parallel using BuildOrchestrator
	buildResults, err := s.buildOrchestrator.BuildAndWait(ctx, projectID, containerPointers)
	if err != nil {
		s.logger.Error(ctx, "build orchestration failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Build orchestration failed: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	// Step 3: Check if all builds succeeded
	if hasFailedBuilds(buildResults) {
		s.logger.Error(ctx, "one or more builds failed",
			zap.Uint("project_id", projectID),
		)
		msg := "One or more container builds failed"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	s.logger.Info(ctx, "all builds completed successfully",
		zap.Uint("project_id", projectID),
	)

	// Step 4: Update container information after successful builds
	// If any update fails, we must abort to avoid deploying with stale metadata
	for i, result := range buildResults {
		if result == nil {
			s.logger.Error(ctx, "build result is nil, aborting post-build updates",
				zap.Uint("project_id", projectID),
				zap.Int("index", i),
			)
			msg := fmt.Sprintf("Build result is nil for container at index %d", i)
			s.handleBuildError(ctx, projectID, &msg)
			return
		}

		err := s.buildPostProcessor.UpdateContainerAfterBuild(
			ctx,
			containerPointers[i].ContainerID,
			result,
			containerPointers[i],
		)
		if err != nil {
			// Check if container changed during build (parameters drift)
			if errors.Is(err, projecterrors.ErrContainerChangedDuringBuild) {
				s.logger.Error(ctx, "container changed during build, cannot proceed with deployment",
					zap.Uint("project_id", projectID),
					zap.Uint("container_id", containerPointers[i].ContainerID),
					zap.String("reason", "parameters changed mid-build, built image is stale"),
					zap.String("mitigation", "needs_build flag remains true, rebuild required"),
				)
				msg := fmt.Sprintf("Container %d changed during build, deployment aborted. Rebuild required.", containerPointers[i].ContainerID)
				s.handleBuildError(ctx, projectID, &msg)
				return
			}

			// Other update errors
			s.logger.Error(ctx, "failed to update container after build, aborting",
				zap.Uint("project_id", projectID),
				zap.Uint("container_id", containerPointers[i].ContainerID),
				zap.Error(err),
			)
			msg := fmt.Sprintf("Failed to update container %d after build: %v", containerPointers[i].ContainerID, err)
			s.handleBuildError(ctx, projectID, &msg)
			return
		}
	}

	s.logger.Info(ctx, "container updates completed, proceeding to deployment",
		zap.Uint("project_id", projectID),
	)

	// Step 5: Proceed to deployment using the snapshot captured before builds started.
	// We use the deployment config snapshot to ensure consistency - it represents the exact
	// state when the user initiated the build+deploy operation.
	// If we fetch it now, we might pick up user edits that happened mid-build,
	// which would be inconsistent with the artifacts we just built.
	s.logger.Info(ctx, "proceeding to deployment",
		zap.Uint("project_id", projectID),
	)

	// Call deployProjectInternal with the captured deployment configuration
	// This method will handle the deployment, monitoring, and project status updates
	if err := s.deployProjectInternal(ctx, projectID, deploymentConfig); err != nil {
		s.logger.Error(ctx, "deployment failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		// deployProjectInternal handles its own cleanup via handleDeployFailure
		return
	}

	s.logger.Info(ctx, "build and deploy background operation completed successfully",
		zap.Uint("project_id", projectID),
	)
}

// handleBuildError handles build failure by resetting project status to 'nothing'
func (s *deployService) handleBuildError(ctx context.Context, projectID uint, summary *string) {
	s.logger.Error(ctx, "handling build error",
		zap.Uint("project_id", projectID),
		zap.Stringp("summary", summary),
	)

	// Create a fresh context with timeout for cleanup operations
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Reset project operation status to 'nothing'
	err := s.txManager.RunInTx(cleanupCtx, func(txCtx context.Context) error {
		proj, err := s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			return fmt.Errorf("failed to find project: %w", err)
		}

		// Only reset if project is in 'building' status
		if proj.OperationStatus() == value.ProjectOperationStatusBuilding {
			if err := proj.CompleteBuild(); err != nil {
				return fmt.Errorf("failed to complete build operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "handleBuildError failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
	} else {
		s.logger.Info(ctx, "project status reset to nothing",
			zap.Uint("project_id", projectID),
		)
	}
}

// hasFailedBuilds checks if any build result indicates failure
func hasFailedBuilds(results []*BuildResult) bool {
	for _, result := range results {
		if result == nil {
			// Nil result is treated as failure
			return true
		}
		// Check for non-success terminal states
		if result.Status != "success" && result.Status != "skipped" {
			return true
		}
	}
	return false
}
