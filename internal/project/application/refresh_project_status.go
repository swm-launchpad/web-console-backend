package application

import (
	"context"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"go.uber.org/zap"
)

// RefreshProjectStatusInput is the input for RefreshProjectStatus use case
type RefreshProjectStatusInput struct {
	ProjectID uint
}

// RefreshProjectStatusUseCase forces a refresh of the project status from Kubernetes
type RefreshProjectStatusUseCase struct {
	projectRepo      repository.ProjectRepository
	deploymentRepo   repository.DeploymentRepository
	buildHistoryRepo repository.BuildHistoryRepository
	containerClient  infrastructure.ContainerClient
	kubeClient       infrastructure.KubeClient
	kubeBuildClient  infrastructure.KubeBuildClient
	logger           logger.Logger
}

// NewRefreshProjectStatusUseCase creates a new RefreshProjectStatusUseCase
func NewRefreshProjectStatusUseCase(
	projectRepo repository.ProjectRepository,
	deploymentRepo repository.DeploymentRepository,
	buildHistoryRepo repository.BuildHistoryRepository,
	containerClient infrastructure.ContainerClient,
	kubeClient infrastructure.KubeClient,
	kubeBuildClient infrastructure.KubeBuildClient,
	log logger.Logger,
) *RefreshProjectStatusUseCase {
	return &RefreshProjectStatusUseCase{
		projectRepo:      projectRepo,
		deploymentRepo:   deploymentRepo,
		buildHistoryRepo: buildHistoryRepo,
		containerClient:  containerClient,
		kubeClient:       kubeClient,
		kubeBuildClient:  kubeBuildClient,
		logger:           log,
	}
}

// Execute forces a refresh of the project status from Kubernetes API
func (uc *RefreshProjectStatusUseCase) Execute(ctx context.Context, input RefreshProjectStatusInput) (*ProjectStatusOutput, error) {
	uc.logger.Info(ctx, "refresh project status started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Load project to check operation_status
	project, err := uc.projectRepo.FindByID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find project",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	output := &ProjectStatusOutput{
		ProjectID:       input.ProjectID,
		OperationStatus: string(project.OperationStatus()),
	}

	// Based on operation_status, refresh appropriate status information
	switch project.OperationStatus() {
	case "building":
		// Get containers for this project
		containers, err := uc.containerClient.GetContainerIDsByProjectID(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get container IDs",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
			return nil, err
		}

		// For each container, get latest build history and refresh if not terminal
		buildStatuses := make([]BuildStatusOutput, 0, len(containers))
		for _, container := range containers {
			buildHistory, err := uc.buildHistoryRepo.FindLatestByContainerID(ctx, container.ContainerID)
			if err != nil {
				// Container may not have any build history yet, skip it
				uc.logger.Warn(ctx, "no build history found for container",
					zap.Uint("container_id", container.ContainerID),
					zap.Error(err),
				)
				continue
			}

			// If build is not completed, refresh from Kubernetes
			if !buildHistory.IsCompleted() {
				uc.logger.Info(ctx, "refreshing non-terminal build from Kubernetes",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Uint("container_id", container.ContainerID),
					zap.String("status", string(buildHistory.Status())),
				)

				refreshedBuildHistory, err := uc.refreshBuildHistoryStatus(ctx, buildHistory)
				if err != nil {
					uc.logger.Error(ctx, "failed to refresh build history status",
						zap.Uint("build_history_id", buildHistory.BuildHistoryID),
						zap.Error(err),
					)
					// Use stale data if refresh fails
				} else {
					buildHistory = refreshedBuildHistory
				}
			}

			// Convert to output DTO
			buildStatus := BuildStatusOutput{
				BuildHistoryID: uint64(buildHistory.BuildHistoryID),
				ContainerID:    container.ContainerID,
				ContainerName:  container.Name,
				Status:         string(buildHistory.Status()),
			}

			// Add optional fields
			if pipelineRun, ok := buildHistory.TektonPipelineRunName(); ok {
				buildStatus.TektonPipelineRun = pipelineRun
			}

			if commitHash, ok := buildHistory.GitCommitHash(); ok {
				buildStatus.GitCommitHash = commitHash
			}

			if startedAt, ok := buildHistory.StartedAt(); ok {
				buildStatus.StartedAt = startedAt.UTC().Format(time.RFC3339)
			}

			if finishedAt, ok := buildHistory.FinishedAt(); ok {
				buildStatus.FinishedAt = finishedAt.UTC().Format(time.RFC3339)
			}

			if summary, ok := buildHistory.Summary(); ok {
				buildStatus.ErrorMessage = summary
			}

			buildStatuses = append(buildStatuses, buildStatus)
		}

		output.BuildStatuses = buildStatuses

	case "deploying":
		// Get active deployment and refresh from Kubernetes
		activeDeploymentID, hasActive := project.ActiveDeploymentID()
		if hasActive {
			deployment, err := uc.deploymentRepo.FindByID(ctx, activeDeploymentID)
			if err != nil {
				uc.logger.Error(ctx, "failed to find active deployment",
					zap.Uint("project_id", input.ProjectID),
					zap.Uint("deployment_id", activeDeploymentID),
					zap.Error(err),
				)
				return nil, err
			}

			// Refresh deployment from Kubernetes if not completed
			if !deployment.IsCompleted() {
				uc.logger.Info(ctx, "refreshing non-terminal deployment from Kubernetes",
					zap.Uint("deployment_id", deployment.DeploymentID),
					zap.String("status", string(deployment.Status())),
				)

				refreshedDeployment, err := uc.refreshDeploymentStatus(ctx, deployment)
				if err != nil {
					uc.logger.Error(ctx, "failed to refresh deployment status",
						zap.Uint("deployment_id", deployment.DeploymentID),
						zap.Error(err),
					)
					// Use stale data if refresh fails
				} else {
					deployment = refreshedDeployment
				}
			}

			// Convert to output DTO
			deploymentStatus := &DeploymentStatusOutput{
				DeploymentID: uint64(deployment.DeploymentID),
				ProjectID:    uint(deployment.ProjectID()),
				Status:       string(deployment.Status()),
				CreatedAt:    deployment.CreatedAt().UTC().Format(time.RFC3339),
			}

			// Add optional fields
			if eventID, ok := deployment.TektonEventID(); ok {
				deploymentStatus.TektonEventID = eventID
			}

			if runName, ok := deployment.TektonPipelineRunName(); ok {
				deploymentStatus.TektonPipelineRunName = runName
			}

			if summary, ok := deployment.Summary(); ok {
				deploymentStatus.Summary = summary
			}

			if startedAt, ok := deployment.StartedAt(); ok {
				deploymentStatus.StartedAt = startedAt.UTC().Format(time.RFC3339)
			}

			if finishedAt, ok := deployment.FinishedAt(); ok {
				deploymentStatus.FinishedAt = finishedAt.UTC().Format(time.RFC3339)
			}

			output.DeploymentStatus = deploymentStatus
		}

	case "nothing":
		// No active operation, nothing to refresh

	default:
		// Unknown status
		uc.logger.Warn(ctx, "unknown operation status",
			zap.Uint("project_id", input.ProjectID),
			zap.String("operation_status", string(project.OperationStatus())),
		)
	}

	uc.logger.Info(ctx, "refresh project status completed",
		zap.Uint("project_id", input.ProjectID),
		zap.String("operation_status", output.OperationStatus),
	)

	return output, nil
}

// refreshBuildHistoryStatus refreshes a build history from Kubernetes API
func (uc *RefreshProjectStatusUseCase) refreshBuildHistoryStatus(
	ctx context.Context,
	bh *build_history.BuildHistory,
) (*build_history.BuildHistory, error) {
	pipelineRunName, hasRunName := bh.TektonPipelineRunName()

	// If no pipeline run name yet, try to find it via label-based lookup
	if !hasRunName {
		// Need TektonEventID to perform lookup
		eventID, hasEventID := bh.TektonEventID()
		if !hasEventID {
			// Critical data inconsistency: build history has no event ID
			uc.logger.Error(ctx, "build history has no Tekton EventID",
				zap.Uint("build_history_id", bh.BuildHistoryID),
				zap.Uint("container_id", bh.ContainerID()),
			)
			msg := "Build history has no Tekton EventID (critical data inconsistency)"
			if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := uc.buildHistoryRepo.Save(ctx, bh); err != nil {
				return nil, err
			}
			return bh, nil
		}

		// Find pipeline run by event ID
		var err error
		pipelineRunName, err = uc.kubeBuildClient.FindPipelineRunNameByEventID(ctx, eventID)
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
					if err := uc.buildHistoryRepo.Save(ctx, bh); err != nil {
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
			if err := uc.buildHistoryRepo.Save(ctx, bh); err != nil {
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
	status, err := uc.kubeBuildClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// If not found, mark as tracking failed (pipeline run deleted)
		if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
			msg := "PipelineRun not found in Kubernetes (deleted)"
			if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := uc.buildHistoryRepo.Save(ctx, bh); err != nil {
				return nil, err
			}
			return bh, nil
		}

		// Other errors (connection/authentication) are retriable
		msg := "Failed to get PipelineRun status (transient error)"
		if err := bh.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingLost, &msg); err != nil {
			return nil, err
		}
		if err := uc.buildHistoryRepo.Save(ctx, bh); err != nil {
			return nil, err
		}
		return bh, nil
	}

	// Update build history status based on PipelineRun status
	if err := uc.updateBuildHistoryFromKubeStatus(ctx, bh, status); err != nil {
		return nil, err
	}

	return bh, nil
}

// refreshDeploymentStatus refreshes a deployment from Kubernetes API
func (uc *RefreshProjectStatusUseCase) refreshDeploymentStatus(
	ctx context.Context,
	d *deployment.Deployment,
) (*deployment.Deployment, error) {
	// If deployment is already completed, no need to refresh
	if d.IsCompleted() {
		return d, nil
	}

	pipelineRunName, hasRunName := d.TektonPipelineRunName()

	// If no pipeline run name yet, try to find it via label-based lookup
	if !hasRunName {
		// Need TektonEventID to perform lookup
		eventID, hasEventID := d.TektonEventID()
		if !hasEventID {
			// Critical data inconsistency: deployment has no event ID
			uc.logger.Error(ctx, "deployment has no Tekton EventID",
				zap.Uint("deployment_id", d.DeploymentID),
				zap.Uint("project_id", d.ProjectID()),
			)
			msg := "Deployment has no Tekton EventID (critical data inconsistency)"
			if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := uc.deploymentRepo.Save(ctx, d); err != nil {
				return nil, err
			}
			return d, nil
		}

		// Find pipeline run by event ID
		var err error
		pipelineRunName, err = uc.kubeClient.FindPipelineRunNameByEventID(ctx, eventID)
		if err != nil {
			// Check if the error is a "not found" error or a transient connectivity/auth issue
			if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
				// PipelineRun truly does not exist - only mark as terminal failure after 5 minute grace period
				createdAt := d.CreatedAt()
				if time.Since(createdAt) > 5*time.Minute {
					msg := "PipelineRun not found after 5 minutes"
					if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingFailed, &msg); err != nil {
						return nil, err
					}
					if err := uc.deploymentRepo.Save(ctx, d); err != nil {
						return nil, err
					}
				}
				return d, nil
			}

			// Other errors (connection/authentication issues) are transient and retriable
			msg := "Failed to find PipelineRun by EventID (transient error)"
			if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingLost, &msg); err != nil {
				return nil, err
			}
			if err := uc.deploymentRepo.Save(ctx, d); err != nil {
				return nil, err
			}
			return d, nil
		}

		// Update deployment with pipeline run name
		runName := pipelineRunName
		if err := d.InitTektonInfo(nil, &runName); err != nil {
			return nil, err
		}
	}

	// Query Kubernetes for current status
	status, err := uc.kubeClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// If not found, mark as tracking failed (pipeline run deleted)
		if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
			msg := "PipelineRun not found in Kubernetes (deleted)"
			if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingFailed, &msg); err != nil {
				return nil, err
			}
			if err := uc.deploymentRepo.Save(ctx, d); err != nil {
				return nil, err
			}
			return d, nil
		}

		// Other errors (connection/authentication) are retriable
		msg := "Failed to get PipelineRun status (transient error)"
		if err := d.UpdateBackendStatus(deployment.DeploymentStatusBackendTrackingLost, &msg); err != nil {
			return nil, err
		}
		if err := uc.deploymentRepo.Save(ctx, d); err != nil {
			return nil, err
		}
		return d, nil
	}

	// Update deployment status based on PipelineRun status
	if err := uc.updateDeploymentFromKubeStatus(ctx, d, status); err != nil {
		return nil, err
	}

	return d, nil
}

// updateDeploymentFromKubeStatus updates deployment based on Kubernetes PipelineRun status
func (uc *RefreshProjectStatusUseCase) updateDeploymentFromKubeStatus(
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

		// Save deployment
		return uc.deploymentRepo.Save(ctx, d)
	}

	if status.Status == "True" {
		// PipelineRun succeeded
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
				return err
			}
		}

		return uc.deploymentRepo.Save(ctx, d)
	}

	if status.Status == "False" {
		// PipelineRun failed or was cancelled
		// Check reason/message to distinguish between failure and cancellation
		var deploymentStatus deployment.DeploymentStatus
		reason := status.Reason
		message := status.Message
		if (reason != "" && (reason == "Cancelled" || reason == "PipelineRunCancelled")) ||
			(message != "" && (message == "PipelineRun cancelled" || message == "TaskRun cancelled")) {
			deploymentStatus = deployment.DeploymentStatusCancelled
		} else {
			deploymentStatus = deployment.DeploymentStatusFailed
		}

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
				return err
			}
		}

		return uc.deploymentRepo.Save(ctx, d)
	}

	// For unknown statuses, just save deployment
	return uc.deploymentRepo.Save(ctx, d)
}

// updateBuildHistoryFromKubeStatus updates build history based on Kubernetes PipelineRun status
func (uc *RefreshProjectStatusUseCase) updateBuildHistoryFromKubeStatus(
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

		return uc.buildHistoryRepo.Save(ctx, bh)
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

		return uc.buildHistoryRepo.Save(ctx, bh)
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

		return uc.buildHistoryRepo.Save(ctx, bh)
	}

	// For unknown statuses, just save build history
	return uc.buildHistoryRepo.Save(ctx, bh)
}
