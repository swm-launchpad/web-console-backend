package build

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// MockBuildOrchestrator is a mock implementation of BuildOrchestrator for testing
type MockBuildOrchestrator struct {
	BuildAndWaitFunc func(ctx context.Context, projectID uint, containers []*dto.BuildContainerInfo) ([]*BuildResult, error)
}

// BuildAndWait implements BuildOrchestrator.BuildAndWait
func (m *MockBuildOrchestrator) BuildAndWait(
	ctx context.Context,
	projectID uint,
	containers []*dto.BuildContainerInfo,
) ([]*BuildResult, error) {
	if m.BuildAndWaitFunc != nil {
		return m.BuildAndWaitFunc(ctx, projectID, containers)
	}

	// Default behavior: return success for all containers
	results := make([]*BuildResult, len(containers))
	for i, container := range containers {
		results[i] = &BuildResult{
			BuildHistoryID:   uint(i + 1),
			ContainerID:      container.ContainerID,
			Status:           "success",
			LatestCommitHash: "abc123def456",
			ImageTag:         "latest",
			ShouldBuild:      container.NeedsBuild,
		}
	}

	return results, nil
}
