package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

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
			// No EventID yet - check if within grace period
			createdAt := d.CreatedAt()
			if time.Since(createdAt) <= 5*time.Minute {
				// Within 5-minute grace period - EventID may not be set yet
				s.logger.Warn(ctx, "Deployment has no EventID but within grace period",
					zap.Uint64("deployment_id", deploymentID),
					zap.Uint("project_id", d.ProjectID()),
					zap.Time("created_at", createdAt),
					zap.Duration("elapsed", time.Since(createdAt)),
				)
				return d, nil
			}

			// Grace period expired - CRITICAL DATA INCONSISTENCY
			// Deployment is not completed but has no EventID - monitoring impossible
			// Project is stuck in 'deploying' state with no recovery path
			s.logger.Error(ctx, "CRITICAL DATA INCONSISTENCY DETECTED (RefreshDeploymentStatus)",
				zap.Uint64("deployment_id", deploymentID),
				zap.Uint("project_id", d.ProjectID()),
				zap.String("status", string(d.Status())),
				zap.Time("created_at", createdAt),
				zap.Duration("elapsed", time.Since(createdAt)),
				zap.String("problem", "Deployment has NO Tekton EventID after grace period"),
				zap.String("impact", "Cannot refresh status - project stuck in 'deploying' state"),
				zap.String("action", "Emergency rollback - setting project->nothing, deployment->backend_tracking_failed"),
			)

			msg := "Deployment refresh impossible: no Tekton EventID found after 5 minutes (critical data inconsistency)"
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

// RefreshActiveBuildStatuses refreshes all build statuses for a project from Kubernetes
func (s *deployService) RefreshActiveBuildStatuses(ctx context.Context, projectID uint) ([]*build_history.BuildHistory, bool, error) {
	s.logger.Info(ctx, "refresh active build statuses started",
		zap.Uint("project_id", projectID),
	)

	// Get containers for this project
	containers, err := s.containerClient.GetContainerIDsByProjectID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container IDs",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, false, err
	}

	// For each container, get latest build history and refresh if not terminal
	buildHistories := make([]*build_history.BuildHistory, 0, len(containers))
	for _, container := range containers {
		buildHistory, err := s.buildHistoryRepo.FindLatestByContainerID(ctx, container.ContainerID)
		if err != nil {
			// Container may not have any build history yet, skip it
			s.logger.Warn(ctx, "no build history found for container",
				zap.Uint("container_id", container.ContainerID),
				zap.Error(err),
			)
			continue
		}

		// If build is not completed, refresh from Kubernetes
		if !buildHistory.IsCompleted() {
			s.logger.Info(ctx, "refreshing non-terminal build from Kubernetes",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.Uint("container_id", container.ContainerID),
				zap.String("status", string(buildHistory.Status())),
			)

			refreshedBuildHistory, err := s.refreshBuildHistoryStatus(ctx, buildHistory)
			if err != nil {
				s.logger.Error(ctx, "failed to refresh build history status",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
				// Use stale data if refresh fails
			} else {
				buildHistory = refreshedBuildHistory
			}
		}

		buildHistories = append(buildHistories, buildHistory)
	}

	// Check if all builds are completed - if so, reset project status to 'nothing'
	// This prevents projects from being stuck in 'building' state if the background goroutine crashes
	allCompleted := true
	for _, bh := range buildHistories {
		status := bh.Status()
		if status != build_history.BuildHistoryStatusSuccess &&
			status != build_history.BuildHistoryStatusSkipped &&
			status != build_history.BuildHistoryStatusFailed &&
			status != build_history.BuildHistoryStatusCancelled &&
			status != build_history.BuildHistoryStatusBackendTrackingFailed {
			allCompleted = false
			break
		}
	}

	projectReset := false
	if allCompleted && len(buildHistories) > 0 {
		// Find the latest finished_at among all build histories
		var latestFinishedAt time.Time
		for _, bh := range buildHistories {
			if finishedAt, ok := bh.FinishedAt(); ok {
				if latestFinishedAt.IsZero() || finishedAt.After(latestFinishedAt) {
					latestFinishedAt = finishedAt
				}
			}
		}

		// 5-minute grace period: Don't reset project status if builds finished recently
		// This prevents race condition where builds complete but deployment hasn't started yet
		if !latestFinishedAt.IsZero() && time.Since(latestFinishedAt) <= 5*time.Minute {
			s.logger.Info(ctx, "skipping project status reset - within grace period",
				zap.Uint("project_id", projectID),
				zap.Time("latest_finished_at", latestFinishedAt),
				zap.Duration("elapsed", time.Since(latestFinishedAt)),
			)
			return buildHistories, false, nil
		}

		s.logger.Info(ctx, "all builds completed and grace period expired, resetting project status to nothing",
			zap.Uint("project_id", projectID),
			zap.Time("latest_finished_at", latestFinishedAt),
			zap.Duration("elapsed", time.Since(latestFinishedAt)),
		)

		// Reset project status using FindByIDForUpdate to prevent race conditions
		proj, err := s.projectRepo.FindByIDForUpdate(ctx, projectID)
		if err != nil {
			s.logger.Error(ctx, "failed to find project for status reset",
				zap.Uint("project_id", projectID),
				zap.Error(err),
			)
			// Don't fail the entire refresh - just log the error
		} else {
			// Only reset if project is still in 'building' status
			if proj.OperationStatus() == value.ProjectOperationStatusBuilding {
				if err := proj.CompleteBuild(); err != nil {
					s.logger.Error(ctx, "failed to complete build operation",
						zap.Uint("project_id", projectID),
						zap.Error(err),
					)
				} else {
					if err := s.projectRepo.Save(ctx, proj); err != nil {
						s.logger.Error(ctx, "failed to save project after status reset",
							zap.Uint("project_id", projectID),
							zap.Error(err),
						)
					} else {
						s.logger.Info(ctx, "project status reset to nothing",
							zap.Uint("project_id", projectID),
						)
						projectReset = true
					}
				}
			}
		}
	}

	s.logger.Info(ctx, "refresh active build statuses completed",
		zap.Uint("project_id", projectID),
		zap.Int("build_count", len(buildHistories)),
		zap.Bool("project_reset", projectReset),
	)

	return buildHistories, projectReset, nil
}

// refreshBuildHistoryStatus refreshes a build history from Kubernetes API
func (s *deployService) refreshBuildHistoryStatus(
	ctx context.Context,
	bh *build_history.BuildHistory,
) (*build_history.BuildHistory, error) {
	pipelineRunName, hasRunName := bh.TektonPipelineRunName()

	// If no pipeline run name yet, try to find it via label-based lookup
	if !hasRunName {
		// Need TektonEventID to perform lookup
		eventID, hasEventID := bh.TektonEventID()
		if !hasEventID {
			// No EventID yet - check if within grace period
			createdAt := bh.CreatedAt()
			if time.Since(createdAt) <= 5*time.Minute {
				// Within 5-minute grace period - EventID may not be set yet
				s.logger.Warn(ctx, "build history has no EventID but within grace period",
					zap.Uint("build_history_id", bh.BuildHistoryID),
					zap.Uint("container_id", bh.ContainerID()),
					zap.Time("created_at", createdAt),
					zap.Duration("elapsed", time.Since(createdAt)),
				)
				return bh, nil
			}

			// Grace period expired - critical data inconsistency
			s.logger.Error(ctx, "build history has no Tekton EventID after grace period",
				zap.Uint("build_history_id", bh.BuildHistoryID),
				zap.Uint("container_id", bh.ContainerID()),
				zap.Time("created_at", createdAt),
				zap.Duration("elapsed", time.Since(createdAt)),
			)
			msg := "Build history has no Tekton EventID after 5 minutes (critical data inconsistency)"
			if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := s.buildHistoryRepo.Save(ctx, bh); err != nil {
				return nil, err
			}
			return bh, nil
		}

		// Find pipeline run by event ID
		var err error
		pipelineRunName, err = s.kubeBuildClient.FindPipelineRunNameByEventID(ctx, eventID)
		if err != nil {
			// Check if the error is a "not found" error or a transient connectivity/auth issue
			if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
				// PipelineRun truly does not exist - only mark as terminal failure after 5 minute grace period
				createdAt := bh.CreatedAt()
				if time.Since(createdAt) > 5*time.Minute {
					msg := "PipelineRun not found after 5 minutes"
					if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &msg); err != nil {
						return nil, err
					}
					if err := s.buildHistoryRepo.Save(ctx, bh); err != nil {
						return nil, err
					}
				}
				return bh, nil
			}

			// Other errors (connection/authentication issues) are transient and retriable
			msg := "Failed to find PipelineRun by EventID (transient error)"
			if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingLost, &msg); err != nil {
				return nil, err
			}
			if err := s.buildHistoryRepo.Save(ctx, bh); err != nil {
				return nil, err
			}
			return bh, nil
		}

		// Update build history with pipeline run name
		runName := pipelineRunName
		if err := bh.InitTektonInfo(nil, &runName); err != nil {
			return nil, err
		}
	}

	// Query Kubernetes for current status
	status, err := s.kubeBuildClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// If not found, mark as tracking failed (pipeline run deleted)
		if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
			msg := "PipelineRun not found in Kubernetes (deleted)"
			if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := s.buildHistoryRepo.Save(ctx, bh); err != nil {
				return nil, err
			}
			return bh, nil
		}

		// Other errors (connection/authentication) are retriable
		msg := "Failed to get PipelineRun status (transient error)"
		if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingLost, &msg); err != nil {
			return nil, err
		}
		if err := s.buildHistoryRepo.Save(ctx, bh); err != nil {
			return nil, err
		}
		return bh, nil
	}

	// Update build history status based on PipelineRun status
	if err := s.updateBuildHistoryFromKubeStatus(ctx, bh, status); err != nil {
		return nil, err
	}

	return bh, nil
}

// updateBuildHistoryFromKubeStatus updates build history based on Kubernetes PipelineRun status
func (s *deployService) updateBuildHistoryFromKubeStatus(
	ctx context.Context,
	bh *build_history.BuildHistory,
	status *dto.PipelineRun,
) error {
	// Similar logic to deployment status update

	if status.Status == "Unknown" {
		// PipelineRun is still running or pending
		if bh.Status() != build_history.BuildHistoryStatusRunning {
			// Initialize Tekton PipelineRun name if not set
			if _, exists := bh.TektonPipelineRunName(); !exists && status.Name != "" {
				name := status.Name
				if err := bh.InitTektonInfo(nil, &name); err != nil {
					return err
				}
			}
		}

		// Update to running status
		var summaryPtr *string
		if status.Message != "" {
			summaryPtr = &status.Message
		}
		if err := bh.UpdateRunningStatus(summaryPtr, status.StartTime); err != nil {
			return err
		}

		return s.buildHistoryRepo.Save(ctx, bh)
	}

	if status.Status == "True" {
		// PipelineRun succeeded
		if !bh.IsCompleted() {
			var summaryPtr *string
			if status.Message != "" {
				summaryPtr = &status.Message
			}
			// Check if build was skipped
			buildStatus := build_history.BuildHistoryStatusSuccess
			if status.Reason == "Skipped" {
				buildStatus = build_history.BuildHistoryStatusSkipped
			}

			finishedAt := time.Now()
			if status.CompletionTime != nil {
				finishedAt = *status.CompletionTime
			}

			// Parse git commit hash from results if available
			var gitCommitHashPtr *string
			// TODO: Parse from status.Results if available

			if err := bh.UpdateCompleteStatus(buildStatus, summaryPtr, gitCommitHashPtr, finishedAt); err != nil {
				return err
			}
		}

		return s.buildHistoryRepo.Save(ctx, bh)
	}

	if status.Status == "False" {
		// PipelineRun failed or was cancelled
		var buildStatus build_history.BuildHistoryStatus
		reason := status.Reason
		message := status.Message
		if (reason != "" && (reason == "Cancelled" || reason == "PipelineRunCancelled")) ||
			(message != "" && (message == "PipelineRun cancelled" || message == "TaskRun cancelled")) {
			buildStatus = build_history.BuildHistoryStatusCancelled
		} else {
			buildStatus = build_history.BuildHistoryStatusFailed
		}

		if !bh.IsCompleted() {
			var summaryPtr *string
			if status.Message != "" {
				summaryPtr = &status.Message
			}
			finishedAt := time.Now()
			if status.CompletionTime != nil {
				finishedAt = *status.CompletionTime
			}

			if err := bh.UpdateCompleteStatus(buildStatus, summaryPtr, nil, finishedAt); err != nil {
				return err
			}
		}

		return s.buildHistoryRepo.Save(ctx, bh)
	}

	// For unknown statuses, just save build history
	return s.buildHistoryRepo.Save(ctx, bh)
}
