package deploy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
)

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

	// Step 2.5: Validate that all containers have been built at least once
	// This is a standalone deploy (not BuildAndDeploy), so we must ensure
	// that all containers have a valid image tag. If any container has never
	// been built, fail fast with a clear validation error instead of allowing
	// Tekton to fail later.
	for _, container := range containerConfig.Containers {
		if container.ImageTag == "pending" {
			s.logger.Error(ctx, "cannot deploy unbuilt container in standalone deploy",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", d.DeploymentID),
				zap.String("container_name", container.Name),
				zap.String("image_tag", container.ImageTag),
			)
			msg := fmt.Sprintf("Container '%s' has never been built (imageTag='pending'). Use build+deploy API instead of standalone deploy.", container.Name)
			s.handleDeployFailure(ctx, projectID, d.DeploymentID, deployment.DeploymentStatusBackendTriggerFailed, &msg)
			return nil, fmt.Errorf("container '%s' has never been built", container.Name)
		}
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
		Namespace:    s.applicationNamespace,
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
		// ImageName is already fully-qualified from ContainerClient
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

		// Verify project is in 'building' status for deployment
		// Only BuildAndDeployProject flow is supported (building -> deploying transition)
		// Standalone deploy (nothing -> deploying) is no longer allowed
		if proj.OperationStatus() != value.ProjectOperationStatusBuilding {
			s.logger.Error(ctx, "deployment must be triggered from building status",
				zap.Uint("project_id", projectID),
				zap.String("current_status", string(proj.OperationStatus())),
				zap.String("expected_status", "building"),
			)
			return fmt.Errorf("invalid operation status for deployment: %s (expected: building)", proj.OperationStatus())
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
		// Transaction failed before deployment was created - reset project status
		// The project is still in 'building' status from buildAndDeployInBackground
		// We must reset it to 'nothing' to avoid leaving it stuck
		msg := fmt.Sprintf("Failed to create deployment in transaction: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
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
				// Deployment completed - check if it succeeded or failed
				// Note: refreshDeploymentStatus already handled project status reset
				if refreshedDeployment.Status() == deployment.DeploymentStatusSuccess {
					return nil
				}
				// Deployment failed, cancelled, or tracking failed
				return fmt.Errorf("deployment completed with status: %s", refreshedDeployment.Status())
			}
		}
	}
}

// BuildAndDeployProject builds all containers for a project, then deploys them
