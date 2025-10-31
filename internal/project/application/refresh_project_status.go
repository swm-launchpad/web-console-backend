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
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/deploy"
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
	deployService    deploy.Deployer
	logger           logger.Logger
}

// NewRefreshProjectStatusUseCase creates a new RefreshProjectStatusUseCase
func NewRefreshProjectStatusUseCase(
	projectRepo repository.ProjectRepository,
	deploymentRepo repository.DeploymentRepository,
	buildHistoryRepo repository.BuildHistoryRepository,
	containerClient infrastructure.ContainerClient,
	deployService deploy.Deployer,
	log logger.Logger,
) *RefreshProjectStatusUseCase {
	return &RefreshProjectStatusUseCase{
		projectRepo:      projectRepo,
		deploymentRepo:   deploymentRepo,
		buildHistoryRepo: buildHistoryRepo,
		containerClient:  containerClient,
		deployService:    deployService,
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
		// Delegate to Domain Service - it handles all refresh logic including project status reset
		buildHistories, projectReset, err := uc.deployService.RefreshActiveBuildStatuses(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to refresh build statuses",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
			return nil, err
		}

		// Get all containers for this project to ensure we return status for all
		containers, err := uc.containerClient.GetContainerIDsByProjectID(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get container IDs",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
			// Continue without build statuses rather than failing the entire request
			containers = []dto.ContainerBasicInfo{}
		}

		// Create a map of buildHistories by container ID for quick lookup
		buildHistoryMap := make(map[uint]*build_history.BuildHistory)
		for _, bh := range buildHistories {
			buildHistoryMap[bh.ContainerID()] = bh
		}

		// Build status output for all containers
		buildStatuses := make([]BuildStatusOutput, 0, len(containers))
		for _, container := range containers {
			buildHistory, exists := buildHistoryMap[container.ContainerID]
			if !exists {
				// No build history for this container
				// If project is building but no history exists yet, return 'running' status
				// Otherwise return 'untracked'
				status := "untracked"
				if project.OperationStatus() == "building" {
					status = "running"
				}
				buildStatus := BuildStatusOutput{
					BuildHistoryID: 0,
					ContainerID:    container.ContainerID,
					ContainerName:  container.Name,
					Status:         status,
				}
				buildStatuses = append(buildStatuses, buildStatus)
				continue
			}

			buildStatus := BuildStatusOutput{
				BuildHistoryID: uint64(buildHistory.BuildHistoryID),
				ContainerID:    buildHistory.ContainerID(),
				ContainerName:  container.Name,
				Status:         string(buildHistory.Status()),
			}

			// Add optional fields
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
				buildStatus.Summary = summary
			}

			buildStatuses = append(buildStatuses, buildStatus)
		}

		output.BuildStatuses = buildStatuses

		// Update output status if project was reset
		if projectReset {
			output.OperationStatus = "nothing"
		}

	case "deploying":
		// Delegate to Domain Service - it handles all refresh logic including project status reset
		deployment, err := uc.deployService.RefreshActiveDeployment(ctx, input.ProjectID)
		if err != nil {
			// If no active deployment, it's OK - project might have just completed
			if errors.Is(err, projecterrors.ErrDeploymentNotFound) {
				uc.logger.Info(ctx, "no active deployment found during refresh",
					zap.Uint("project_id", input.ProjectID),
				)
			} else {
				uc.logger.Error(ctx, "failed to refresh deployment",
					zap.Uint("project_id", input.ProjectID),
					zap.Error(err),
				)
				return nil, err
			}
		}

		if deployment != nil {
			// Convert to output DTO
			deploymentStatus := &DeploymentStatusOutput{
				DeploymentID: uint64(deployment.DeploymentID),
				ProjectID:    uint(deployment.ProjectID()),
				Status:       string(deployment.Status()),
			}

			// Add optional fields
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

		// Also return build statuses to maintain "last operation" date consistency
		containers, err := uc.containerClient.GetContainerIDsByProjectID(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get container IDs",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
			// Continue without build statuses rather than failing the entire request
			containers = []dto.ContainerBasicInfo{}
		}

		// Get latest build history for each container
		buildStatuses := make([]BuildStatusOutput, 0, len(containers))
		for _, container := range containers {
			buildHistory, err := uc.buildHistoryRepo.FindLatestByContainerID(ctx, container.ContainerID)
			if err != nil {
				// Container may not have any build history yet
				uc.logger.Warn(ctx, "no build history found for container",
					zap.Uint("container_id", container.ContainerID),
					zap.Error(err),
				)
				// Return untracked status for containers without build history
				buildStatus := BuildStatusOutput{
					BuildHistoryID: 0,
					ContainerID:    container.ContainerID,
					ContainerName:  container.Name,
					Status:         "untracked",
				}
				buildStatuses = append(buildStatuses, buildStatus)
				continue
			}

			buildStatus := BuildStatusOutput{
				BuildHistoryID: uint64(buildHistory.BuildHistoryID),
				ContainerID:    buildHistory.ContainerID(),
				ContainerName:  container.Name,
				Status:         string(buildHistory.Status()),
			}

			// Add optional fields
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
				buildStatus.Summary = summary
			}

			buildStatuses = append(buildStatuses, buildStatus)
		}

		output.BuildStatuses = buildStatuses

	case "nothing":
		// No active operation - return last build statuses and deployment status
		// This ensures frontend can display the final state after operations complete

		// Get containers and their latest build statuses
		containers, err := uc.containerClient.GetContainerIDsByProjectID(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get container IDs",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
			// Continue without build statuses rather than failing the entire request
			containers = []dto.ContainerBasicInfo{}
		}

		// Get latest build history for each container
		buildStatuses := make([]BuildStatusOutput, 0, len(containers))
		for _, container := range containers {
			buildHistory, err := uc.buildHistoryRepo.FindLatestByContainerID(ctx, container.ContainerID)
			if err != nil {
				// Container may not have any build history yet
				uc.logger.Warn(ctx, "no build history found for container",
					zap.Uint("container_id", container.ContainerID),
					zap.Error(err),
				)
				// Return untracked status for containers without build history
				buildStatus := BuildStatusOutput{
					BuildHistoryID: 0,
					ContainerID:    container.ContainerID,
					ContainerName:  container.Name,
					Status:         "untracked",
				}
				buildStatuses = append(buildStatuses, buildStatus)
				continue
			}

			buildStatus := BuildStatusOutput{
				BuildHistoryID: uint64(buildHistory.BuildHistoryID),
				ContainerID:    buildHistory.ContainerID(),
				ContainerName:  container.Name,
				Status:         string(buildHistory.Status()),
			}

			// Add optional fields
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
				buildStatus.Summary = summary
			}

			buildStatuses = append(buildStatuses, buildStatus)
		}

		output.BuildStatuses = buildStatuses

		// Get latest deployment if exists
		deployment, err := uc.deploymentRepo.FindLatestByProjectID(ctx, input.ProjectID)
		if err != nil {
			// No deployment history is normal for projects that haven't been deployed yet
			uc.logger.Debug(ctx, "no deployment history found",
				zap.Uint("project_id", input.ProjectID),
				zap.Error(err),
			)
		} else {
			deploymentStatus := &DeploymentStatusOutput{
				DeploymentID: uint64(deployment.DeploymentID),
				ProjectID:    uint(deployment.ProjectID()),
				Status:       string(deployment.Status()),
			}

			// Add optional fields
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
