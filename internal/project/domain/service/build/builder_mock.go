package build

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// MockBuildService is a mock implementation of BuildService for testing
type MockBuildService struct {
	BuildContainerFunc func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error)
}

// BuildContainer implements BuildService.BuildContainer
func (m *MockBuildService) BuildContainer(
	ctx context.Context,
	buildHistoryID uint,
	container *dto.BuildContainerInfo,
) (*BuildResult, error) {
	if m.BuildContainerFunc != nil {
		return m.BuildContainerFunc(ctx, buildHistoryID, container)
	}

	// Default behavior: return success
	return &BuildResult{
		BuildHistoryID:   buildHistoryID,
		Status:           "success",
		LatestCommitHash: "abc123def456",
		ImageTag:         "latest",
		ShouldBuild:      true,
	}, nil
}
