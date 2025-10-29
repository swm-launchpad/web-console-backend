package build

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// MockBuildPostProcessor is a mock implementation of BuildPostProcessor for testing
type MockBuildPostProcessor struct {
	UpdateContainerAfterBuildFunc func(ctx context.Context, containerID uint, buildResult *BuildResult, snapshotBeforeBuild *dto.BuildContainerInfo) error
}

// UpdateContainerAfterBuild implements BuildPostProcessor.UpdateContainerAfterBuild
func (m *MockBuildPostProcessor) UpdateContainerAfterBuild(
	ctx context.Context,
	containerID uint,
	buildResult *BuildResult,
	snapshotBeforeBuild *dto.BuildContainerInfo,
) error {
	if m.UpdateContainerAfterBuildFunc != nil {
		return m.UpdateContainerAfterBuildFunc(ctx, containerID, buildResult, snapshotBeforeBuild)
	}

	// Default behavior: return success
	return nil
}
