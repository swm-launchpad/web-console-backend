package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
)

// GetProjectStatusInput is the input for GetProjectStatus use case
type GetProjectStatusInput struct {
	ProjectID uint
}

// ProjectStatusOutput represents the integrated status of a project
type ProjectStatusOutput struct {
	ProjectID        uint                    `json:"project_id"`
	OperationStatus  string                  `json:"operation_status"` // nothing, building, deploying
	BuildStatuses    []BuildStatusOutput     `json:"build_statuses,omitempty"`
	DeploymentStatus *DeploymentStatusOutput `json:"deployment_status,omitempty"`
}

// BuildStatusOutput represents the status of a container build
type BuildStatusOutput struct {
	BuildHistoryID uint64 `json:"build_history_id"`
	ContainerID    uint   `json:"container_id"`
	ContainerName  string `json:"container_name"`
	Status         string `json:"status"`
	Summary        string `json:"summary,omitempty"`
	GitCommitHash  string `json:"git_commit_hash,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
}

// DeploymentStatusOutput represents the status of a deployment
type DeploymentStatusOutput struct {
	DeploymentID uint64 `json:"deployment_id"`
	ProjectID    uint   `json:"project_id"`
	Status       string `json:"status"`
	Summary      string `json:"summary,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

// GetProjectStatusUseCase retrieves the integrated status of a project from the database
type GetProjectStatusUseCase struct {
	projectRepo      repository.ProjectRepository
	deploymentRepo   repository.DeploymentRepository
	buildHistoryRepo repository.BuildHistoryRepository
	containerClient  infrastructure.ContainerClient
	logger           logger.Logger
}

// NewGetProjectStatusUseCase creates a new GetProjectStatusUseCase
func NewGetProjectStatusUseCase(
	projectRepo repository.ProjectRepository,
	deploymentRepo repository.DeploymentRepository,
	buildHistoryRepo repository.BuildHistoryRepository,
	containerClient infrastructure.ContainerClient,
	log logger.Logger,
) *GetProjectStatusUseCase {
	return &GetProjectStatusUseCase{
		projectRepo:      projectRepo,
		deploymentRepo:   deploymentRepo,
		buildHistoryRepo: buildHistoryRepo,
		containerClient:  containerClient,
		logger:           log,
	}
}

// Execute retrieves the integrated status of a project from the database
func (uc *GetProjectStatusUseCase) Execute(ctx context.Context, input GetProjectStatusInput) (*ProjectStatusOutput, error) {
	uc.logger.Info(ctx, "get project status started",
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

	// Based on operation_status, retrieve appropriate status information
	switch project.OperationStatus() {
	case "building":
		// Get containers for this project
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
				ContainerID:    container.ContainerID,
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

		// Also return latest deployment to maintain "last operation" date consistency
		deployment, err := uc.deploymentRepo.FindLatestByProjectID(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Debug(ctx, "no deployment history found during build",
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

	case "deploying":
		// Get active deployment
		activeDeploymentID, hasActive := project.ActiveDeploymentID()
		if hasActive {
			deployment, err := uc.deploymentRepo.FindByID(ctx, activeDeploymentID)
			if err != nil {
				uc.logger.Error(ctx, "failed to find active deployment",
					zap.Uint("project_id", input.ProjectID),
					zap.Uint("deployment_id", activeDeploymentID),
					zap.Error(err),
				)
				// Continue without deployment status rather than failing the entire request
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
				ContainerID:    container.ContainerID,
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
				ContainerID:    container.ContainerID,
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
		// Unknown status, just return the status
		uc.logger.Warn(ctx, "unknown operation status",
			zap.Uint("project_id", input.ProjectID),
			zap.String("operation_status", string(project.OperationStatus())),
		)
	}

	uc.logger.Info(ctx, "get project status completed",
		zap.Uint("project_id", input.ProjectID),
		zap.String("operation_status", output.OperationStatus),
	)

	return output, nil
}
