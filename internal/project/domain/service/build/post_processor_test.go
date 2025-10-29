package build

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// Mock ContainerUpdater for testing

type mockContainerUpdater struct {
	updateAfterBuildFunc func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error)
}

func (m *mockContainerUpdater) UpdateAfterBuild(
	ctx context.Context,
	containerID uint,
	buildStatus string,
	commitHash string,
	snapshot *dto.BuildContainerInfo,
) (bool, error) {
	if m.updateAfterBuildFunc != nil {
		return m.updateAfterBuildFunc(ctx, containerID, buildStatus, commitHash, snapshot)
	}
	return true, nil // Default: return wasUpdated=true
}

// Test cases

func TestBuildPostProcessor_UpdateContainerAfterBuild_Success(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var capturedContainerID uint
	var capturedBuildStatus string
	var capturedCommitHash string
	var capturedSnapshot *dto.BuildContainerInfo

	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			capturedContainerID = containerID
			capturedBuildStatus = buildStatus
			capturedCommitHash = commitHash
			capturedSnapshot = snapshot
			return true, nil // Return wasUpdated=true
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "success",
		LatestCommitHash: "abc123",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	assert.NoError(t, err)
	assert.Equal(t, uint(10), capturedContainerID)
	assert.Equal(t, "success", capturedBuildStatus)
	assert.Equal(t, "abc123", capturedCommitHash)
	assert.Equal(t, snapshot, capturedSnapshot)
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_NonSuccessStatus(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var updaterCalled bool
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			updaterCalled = true
			return false, nil // Return wasUpdated=false for non-success status
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "failed",
		ErrorMessage:     "build failed",
		LatestCommitHash: "",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// UpdateAfterBuild should still be called (it decides whether to update)
	assert.NoError(t, err)
	assert.True(t, updaterCalled)
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_UpdaterError(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	expectedErr := errors.New("update failed")
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			return false, expectedErr
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "success",
		LatestCommitHash: "abc123",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update container after build")
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_SkippedBuild(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var capturedBuildStatus string
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			capturedBuildStatus = buildStatus
			return true, nil // Skipped builds still update (clear needs_build flag)
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "skipped",
		LatestCommitHash: "",
		ShouldBuild:      false,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	assert.NoError(t, err)
	assert.Equal(t, "skipped", capturedBuildStatus)
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_BackendTrackingLost(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var updaterCalled bool
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			updaterCalled = true
			return false, nil // Return wasUpdated=false for non-success status
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "backend_tracking_lost",
		ErrorMessage:     "Context cancelled",
		LatestCommitHash: "",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Should delegate to updater even for non-terminal states
	assert.NoError(t, err)
	assert.True(t, updaterCalled)
}

// TestBuildPostProcessor_UpdateContainerAfterBuild_ParametersChangedDuringBuild tests the snapshot drift scenario
func TestBuildPostProcessor_UpdateContainerAfterBuild_ParametersChangedDuringBuild(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	// Mock: Updater returns wasUpdated=false for success status
	// This simulates container parameters changing mid-build
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			// Return wasUpdated=false to indicate snapshot comparison failed
			return false, nil
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "success", // Build succeeded
		LatestCommitHash: "abc123",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Should return ErrContainerChangedDuringBuild
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrContainerChangedDuringBuild)
}

// TestBuildPostProcessor_UpdateContainerAfterBuild_SkippedBuildWithParametersChanged tests the skipped build drift scenario
func TestBuildPostProcessor_UpdateContainerAfterBuild_SkippedBuildWithParametersChanged(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	// Mock: Updater returns wasUpdated=false for skipped status
	// This simulates container parameters changing during a skipped build
	mockUpdater := &mockContainerUpdater{
		updateAfterBuildFunc: func(ctx context.Context, containerID uint, buildStatus string, commitHash string, snapshot *dto.BuildContainerInfo) (bool, error) {
			// Return wasUpdated=false to indicate snapshot comparison failed
			return false, nil
		},
	}

	processor := NewPostProcessor(mockUpdater, testLogger)

	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "skipped", // Build was skipped by Tekton
		LatestCommitHash: "",
		ShouldBuild:      false,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Should return ErrContainerChangedDuringBuild even for skipped builds
	// because parameters changed mid-flight, making any cached image stale
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrContainerChangedDuringBuild)
}
