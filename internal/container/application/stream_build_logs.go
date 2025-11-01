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

// StreamBuildLogsInput represents the input for streaming build logs
type StreamBuildLogsInput struct {
	ContainerID uint
}

// StreamBuildLogsUseCase handles streaming build logs from Loki
type StreamBuildLogsUseCase struct {
	buildHistoryRepo projectrepo.BuildHistoryRepository
	lokiClient       infrastructure.LokiClient
	logger           logger.Logger
}

// NewStreamBuildLogsUseCase creates a new instance of StreamBuildLogsUseCase
func NewStreamBuildLogsUseCase(
	buildHistoryRepo projectrepo.BuildHistoryRepository,
	lokiClient infrastructure.LokiClient,
	log logger.Logger,
) *StreamBuildLogsUseCase {
	return &StreamBuildLogsUseCase{
		buildHistoryRepo: buildHistoryRepo,
		lokiClient:       lokiClient,
		logger:           log,
	}
}

// Execute streams build logs for a container
// Returns an io.ReadCloser that streams logs from Loki
// The caller is responsible for closing the returned ReadCloser
func (uc *StreamBuildLogsUseCase) Execute(ctx context.Context, input StreamBuildLogsInput) (io.ReadCloser, error) {
	uc.logger.Info(ctx, "Streaming build logs",
		zap.Uint("container_id", input.ContainerID),
	)

	// First, try to find active builds (currently running)
	activeBuilds, err := uc.buildHistoryRepo.FindActiveByContainerID(ctx, input.ContainerID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find active builds",
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	var pipelineRunName string

	// Active builds (untracked, running, backend_tracking_lost) may not have a PipelineRunName yet
	// If no PipelineRunName is available, we cannot stream logs from Loki
	if len(activeBuilds) > 0 {
		activeBuild := activeBuilds[0]
		prName, hasPRName := activeBuild.TektonPipelineRunName()
		if !hasPRName {
			// Active build exists but no PipelineRunName yet (e.g., untracked status)
			uc.logger.Warn(ctx, "Active build exists but has no PipelineRunName",
				zap.Uint("container_id", input.ContainerID),
				zap.Uint("build_history_id", activeBuild.BuildHistoryID),
			)
			return nil, projecterrors.ErrBuildHistoryNotFound
		}

		pipelineRunName = prName
		uc.logger.Info(ctx, "Found active build",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", activeBuild.BuildHistoryID),
			zap.String("pipeline_run_name", pipelineRunName),
		)
	} else {
		// No active build, try to find the latest build
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

		prName, hasPRName := latestBuild.TektonPipelineRunName()
		if !hasPRName {
			uc.logger.Warn(ctx, "Latest build has no PipelineRunName",
				zap.Uint("container_id", input.ContainerID),
				zap.Uint("build_history_id", latestBuild.BuildHistoryID),
			)
			return nil, projecterrors.ErrBuildHistoryNotFound
		}

		pipelineRunName = prName
		uc.logger.Info(ctx, "Found latest build",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", latestBuild.BuildHistoryID),
			zap.String("pipeline_run_name", pipelineRunName),
		)
	}

	// Stream logs from Loki, excluding ecr-repository-check task
	excludeTasks := []string{"ecr-repository-check"}
	logStream, err := uc.lokiClient.StreamPipelineRunLogs(ctx, pipelineRunName, excludeTasks)
	if err != nil {
		uc.logger.Error(ctx, "Failed to stream logs from Loki",
			zap.Uint("container_id", input.ContainerID),
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Successfully started streaming build logs",
		zap.Uint("container_id", input.ContainerID),
		zap.String("pipeline_run_name", pipelineRunName),
	)

	return logStream, nil
}
