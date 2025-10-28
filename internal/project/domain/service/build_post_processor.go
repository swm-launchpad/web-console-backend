package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// BuildPostProcessor defines the interface for post-build container updates.
// This service is responsible for updating container state after a build completes,
// including last_built_commit_hash and needs_build flags.
type BuildPostProcessor interface {
	// UpdateContainerAfterBuild updates container information after a build completes.
	// This method delegates to ContainerUpdater which handles:
	//  - Acquiring row lock and comparing build parameter snapshots
	//  - Updating last_built_git_commit_hash and needs_build flags
	//  - Skipping updates if parameters changed during the build
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadline control
	//   - containerID: The unique identifier of the container to update
	//   - buildResult: The result of the build operation
	//   - snapshotBeforeBuild: Container configuration snapshot taken before build started
	//
	// Returns:
	//   - error: Returns error if update fails
	UpdateContainerAfterBuild(
		ctx context.Context,
		containerID uint,
		buildResult *BuildResult,
		snapshotBeforeBuild *dto.BuildContainerInfo,
	) error
}
