package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// buildPostProcessorImpl implements BuildPostProcessor
type buildPostProcessorImpl struct {
	containerUpdater ContainerUpdater
	logger           logger.Logger
}

// NewBuildPostProcessor creates a new BuildPostProcessor instance
func NewBuildPostProcessor(
	containerUpdater ContainerUpdater,
	log logger.Logger,
) BuildPostProcessor {
	return &buildPostProcessorImpl{
		containerUpdater: containerUpdater,
		logger:           log,
	}
}

// UpdateContainerAfterBuild updates container information after a successful build
func (p *buildPostProcessorImpl) UpdateContainerAfterBuild(
	ctx context.Context,
	containerID uint,
	buildResult *BuildResult,
	snapshotBeforeBuild *dto.BuildContainerInfo,
) error {
	p.logger.Info(ctx, "Starting post-build container update",
		zap.Uint("container_id", containerID),
		zap.Uint("build_history_id", buildResult.BuildHistoryID),
		zap.String("build_status", buildResult.Status),
	)

	// Delegate to container updater (dependency inversion principle)
	wasUpdated, err := p.containerUpdater.UpdateAfterBuild(
		ctx,
		containerID,
		buildResult.Status,
		buildResult.LatestCommitHash,
		snapshotBeforeBuild,
	)

	if err != nil {
		return fmt.Errorf("failed to update container after build: %w", err)
	}

	// Log based on whether update was actually performed
	if wasUpdated {
		p.logger.Info(ctx, "Successfully updated container after build",
			zap.Uint("container_id", containerID),
			zap.String("build_status", buildResult.Status),
		)
	} else {
		// Update was skipped (e.g., build parameters changed mid-flight or non-success status)
		p.logger.Info(ctx, "Container update skipped",
			zap.Uint("container_id", containerID),
			zap.String("build_status", buildResult.Status),
			zap.String("reason", "build parameters changed or non-success status"),
		)
	}

	return nil
}
