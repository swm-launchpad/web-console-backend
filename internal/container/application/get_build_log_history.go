package application

import (
	"context"
	"io"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// GetBuildLogHistoryInput represents the input for retrieving historical build logs
type GetBuildLogHistoryInput struct {
	ContainerID uint
}

// GetBuildLogHistoryUseCase handles retrieving historical build logs from Loki
type GetBuildLogHistoryUseCase struct {
	buildHistoryRepo projectrepo.BuildHistoryRepository
	lokiClient       infrastructure.LokiClient
	logger           logger.Logger
}

// NewGetBuildLogHistoryUseCase creates a new instance of GetBuildLogHistoryUseCase
func NewGetBuildLogHistoryUseCase(
	buildHistoryRepo projectrepo.BuildHistoryRepository,
	lokiClient infrastructure.LokiClient,
	log logger.Logger,
) *GetBuildLogHistoryUseCase {
	return &GetBuildLogHistoryUseCase{
		buildHistoryRepo: buildHistoryRepo,
		lokiClient:       lokiClient,
		logger:           log,
	}
}

// Execute retrieves historical build logs for a container's latest completed build
// Returns an io.ReadCloser containing the Loki query_range response
// The caller is responsible for closing the returned ReadCloser
func (uc *GetBuildLogHistoryUseCase) Execute(ctx context.Context, input GetBuildLogHistoryInput) (io.ReadCloser, error) {
	uc.logger.Info(ctx, "Retrieving historical build logs",
		zap.Uint("container_id", input.ContainerID),
	)

	// Find the latest build for this container
	latestBuild, err := uc.buildHistoryRepo.FindLatestByContainerID(ctx, input.ContainerID)
	if err != nil {
		if err == projecterrors.ErrBuildHistoryNotFound {
			uc.logger.Warn(ctx, "No build history found for container",
				zap.Uint("container_id", input.ContainerID),
			)
			return nil, projecterrors.ErrBuildHistoryNotFound
		}
		uc.logger.Error(ctx, "Failed to find latest build",
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	// Verify the build is completed
	if !latestBuild.IsCompleted() {
		uc.logger.Warn(ctx, "Latest build is not completed yet",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", latestBuild.BuildHistoryID),
		)
		return nil, projecterrors.ErrBuildHistoryNotFound
	}

	// Get PipelineRunName
	prName, hasPRName := latestBuild.TektonPipelineRunName()
	if !hasPRName {
		uc.logger.Warn(ctx, "Latest build has no PipelineRunName",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", latestBuild.BuildHistoryID),
		)
		return nil, projecterrors.ErrBuildHistoryNotFound
	}

	// Get build time range
	startedAt, hasStartedAt := latestBuild.StartedAt()
	finishedAt, hasFinishedAt := latestBuild.FinishedAt()

	if !hasStartedAt || !hasFinishedAt {
		uc.logger.Warn(ctx, "Build missing time range information",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", latestBuild.BuildHistoryID),
			zap.Bool("has_started_at", hasStartedAt),
			zap.Bool("has_finished_at", hasFinishedAt),
		)
		return nil, projecterrors.ErrBuildHistoryNotFound
	}

	uc.logger.Info(ctx, "Found completed build for historical logs",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("build_history_id", latestBuild.BuildHistoryID),
		zap.String("pipeline_run_name", prName),
		zap.Time("started_at", startedAt),
		zap.Time("finished_at", finishedAt),
	)

	// Query historical logs from Loki via HTTP, excluding ecr-repository-check task
	excludeTasks := []string{"ecr-repository-check"}
	logData, err := uc.lokiClient.QueryPipelineRunLogsHTTP(ctx, prName, excludeTasks, startedAt, finishedAt)
	if err != nil {
		uc.logger.Error(ctx, "Failed to query logs from Loki",
			zap.Uint("container_id", input.ContainerID),
			zap.String("pipeline_run_name", prName),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Successfully retrieved historical build logs",
		zap.Uint("container_id", input.ContainerID),
		zap.String("pipeline_run_name", prName),
	)

	return logData, nil
}
