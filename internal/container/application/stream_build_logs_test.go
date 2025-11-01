package application

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

func TestStreamBuildLogsUseCase_Execute_ActiveBuild(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewStreamBuildLogsUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		log,
	)

	input := StreamBuildLogsInput{
		ContainerID: 10,
	}

	// Create active build with PipelineRunName
	pipelineRunName := "image-build-push-run-abc123"
	activeBuild, _ := build_history.ReconstructBuildHistory(
		1,
		10,
		build_history.BuildHistoryStatusRunning,
		nil,
		stringPtr("event-123"),
		&pipelineRunName,
		nil,
		time.Now(),
		timePtr(time.Now()),
		nil,
	)

	// Mock FindActiveByContainerID - returns active build
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{activeBuild}, nil)

	// Mock StreamPipelineRunLogs
	mockLogStream := io.NopCloser(strings.NewReader("log data"))
	mockLokiClient.On("StreamPipelineRunLogs", ctx, pipelineRunName, []string{"ecr-repository-check"}).
		Return(mockLogStream, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, logStream)

	// Verify we can read from the stream
	data, _ := io.ReadAll(logStream)
	assert.Equal(t, "log data", string(data))

	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestStreamBuildLogsUseCase_Execute_LatestBuild(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewStreamBuildLogsUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		log,
	)

	input := StreamBuildLogsInput{
		ContainerID: 10,
	}

	// Create latest build with PipelineRunName
	pipelineRunName := "image-build-push-run-xyz789"
	latestBuild, _ := build_history.ReconstructBuildHistory(
		2,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-456"),
		&pipelineRunName,
		nil,
		time.Now().Add(-1*time.Hour),
		timePtr(time.Now().Add(-1*time.Hour)),
		timePtr(time.Now().Add(-30*time.Minute)),
	)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Mock FindLatestByContainerID - returns completed build
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(latestBuild, nil)

	// Mock StreamPipelineRunLogs
	mockLogStream := io.NopCloser(strings.NewReader("completed log data"))
	mockLokiClient.On("StreamPipelineRunLogs", ctx, pipelineRunName, []string{"ecr-repository-check"}).
		Return(mockLogStream, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, logStream)

	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestStreamBuildLogsUseCase_Execute_NoBuildHistory(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewStreamBuildLogsUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		log,
	)

	input := StreamBuildLogsInput{
		ContainerID: 999, // Non-existent container
	}

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(999)).
		Return([]*build_history.BuildHistory{}, nil)

	// Mock FindLatestByContainerID - no build history
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(999)).
		Return(nil, projecterrors.ErrBuildHistoryNotFound)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logStream)

	mockBuildHistoryRepo.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "StreamPipelineRunLogs")
}

func TestStreamBuildLogsUseCase_Execute_NoPipelineRunName(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewStreamBuildLogsUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		log,
	)

	input := StreamBuildLogsInput{
		ContainerID: 10,
	}

	// Create latest build WITHOUT PipelineRunName (untracked status)
	latestBuild, _ := build_history.ReconstructBuildHistory(
		3,
		10,
		build_history.BuildHistoryStatusUntracked,
		nil,
		stringPtr("event-789"),
		nil, // No PipelineRunName
		nil,
		time.Now().Add(-1*time.Hour),
		nil,
		nil,
	)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Mock FindLatestByContainerID - returns build without PipelineRunName
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(latestBuild, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logStream)

	mockBuildHistoryRepo.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "StreamPipelineRunLogs")
}

func TestStreamBuildLogsUseCase_Execute_ActiveBuildWithoutPipelineRunName(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewStreamBuildLogsUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		log,
	)

	input := StreamBuildLogsInput{
		ContainerID: 10,
	}

	// Create active build WITHOUT PipelineRunName (e.g., untracked status)
	activeBuild, _ := build_history.ReconstructBuildHistory(
		4,
		10,
		build_history.BuildHistoryStatusUntracked,
		nil,
		stringPtr("event-abc"),
		nil, // No PipelineRunName yet
		nil,
		time.Now(),
		nil,
		nil,
	)

	// Mock FindActiveByContainerID - returns active build without PipelineRunName
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{activeBuild}, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert - should fail because active build has no PipelineRunName
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logStream)

	mockBuildHistoryRepo.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "StreamPipelineRunLogs")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
