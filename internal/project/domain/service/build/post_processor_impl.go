package build

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/build/adapter"
)

// postProcessorImpl implements PostProcessor
type postProcessorImpl struct {
	containerUpdater adapter.ContainerUpdater
	logger           logger.Logger
}

// NewPostProcessor creates a new PostProcessor instance
func NewPostProcessor(
	containerUpdater adapter.ContainerUpdater,
	log logger.Logger,
) PostProcessor {
	return &postProcessorImpl{
		containerUpdater: containerUpdater,
		logger:           log,
	}
}

// UpdateContainerAfterBuild updates container information after a successful build
func (p *postProcessorImpl) UpdateContainerAfterBuild(
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

	// Critical: If build completed (success or skipped) but update was skipped, parameters changed mid-build.
	// This means the built image is now stale and should not be deployed.
	// needs_build flag remains true, which is our signal that the image is stale.
	// Note: "skipped" builds can also have parameter drift - Tekton skips the build but parameters may have changed,
	// leading to the same stale-image deployment risk once PR#10 wires in the deploy step.
	if (buildResult.Status == "success" || buildResult.Status == "skipped") && !wasUpdated {
		p.logger.Error(ctx, "Container parameters changed during build, aborting deployment",
			zap.Uint("container_id", containerID),
			zap.String("build_status", buildResult.Status),
			zap.String("reason", "snapshot comparison failed - parameters changed mid-build"),
		)
		return projecterrors.ErrContainerChangedDuringBuild
	}

	// Log based on whether update was actually performed
	if wasUpdated {
		p.logger.Info(ctx, "Successfully updated container after build",
			zap.Uint("container_id", containerID),
			zap.String("build_status", buildResult.Status),
		)
	} else {
		// Build failed - update correctly not performed
		p.logger.Info(ctx, "Container update skipped for non-success build",
			zap.Uint("container_id", containerID),
			zap.String("build_status", buildResult.Status),
			zap.String("reason", "build did not succeed"),
		)
	}

	return nil
}
