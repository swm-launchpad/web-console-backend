package application

import (
	"context"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/deploy"
	"go.uber.org/zap"
)

// RefreshProjectStatusInput is the input for RefreshProjectStatus use case
type RefreshProjectStatusInput struct {
	ProjectID uint
}

// RefreshProjectStatusUseCase forces a refresh of the project status from Kubernetes
type RefreshProjectStatusUseCase struct {
	projectRepo   repository.ProjectRepository
	deployService deploy.Deployer
	logger        logger.Logger
}

// NewRefreshProjectStatusUseCase creates a new RefreshProjectStatusUseCase
func NewRefreshProjectStatusUseCase(
	projectRepo repository.ProjectRepository,
	deployService deploy.Deployer,
	log logger.Logger,
) *RefreshProjectStatusUseCase {
	return &RefreshProjectStatusUseCase{
		projectRepo:   projectRepo,
		deployService: deployService,
		logger:        log,
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

		// Convert to output DTOs
		buildStatuses := make([]BuildStatusOutput, 0, len(buildHistories))
		for _, buildHistory := range buildHistories {
			buildStatus := BuildStatusOutput{
				BuildHistoryID: uint64(buildHistory.BuildHistoryID),
				ContainerID:    buildHistory.ContainerID(),
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
