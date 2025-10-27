package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// BuildPostProcessor defines the interface for post-build container updates.
// This service is responsible for updating container state after a build completes,
// including last_built_commit_hash and needs_build flags.
type BuildPostProcessor interface {
	// UpdateContainerAfterBuild updates container information after a successful build.
	// This method:
	//  1. Acquires a row lock (FOR UPDATE) on the container record
	//  2. Compares the snapshot taken before build with current database state
	//  3. If build parameters haven't changed during the build:
	//     - Updates last_built_git_commit_hash from BuildResult
	//     - Sets needs_build = false
	//     - Updates updated_at timestamp
	//  4. If build parameters changed during the build:
	//     - Skips update (needs_build remains true for next build)
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadline control
	//   - containerID: The unique identifier of the container to update
	//   - buildResult: The result of the build operation
	//   - snapshotBeforeBuild: Container configuration snapshot taken before build started
	//
	// Returns:
	//   - error: Returns error if update fails or lock cannot be acquired
	//
	// Note: This method should be called within a transaction to ensure consistency
	UpdateContainerAfterBuild(
		ctx context.Context,
		containerID uint,
		buildResult *BuildResult,
		snapshotBeforeBuild *dto.BuildContainerInfo,
	) error
}
