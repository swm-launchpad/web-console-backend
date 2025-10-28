package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// ContainerUpdater defines the interface for updating containers after builds
// This interface allows the domain layer to remain independent of infrastructure details
// while still being able to update container state after build completion
type ContainerUpdater interface {
	// UpdateAfterBuild updates a container's state after a build completes
	// It compares the snapshot taken before the build with the current state to detect changes
	// and only updates if no build parameters have changed during the build process
	//
	// Returns:
	//   - wasUpdated: true if the container was actually updated, false if update was skipped
	//                 (e.g., due to build parameters changing mid-flight)
	//   - error: any error that occurred during the update process
	UpdateAfterBuild(
		ctx context.Context,
		containerID uint,
		buildStatus string,
		commitHash string,
		snapshotBeforeBuild *dto.BuildContainerInfo,
	) (wasUpdated bool, err error)
}
