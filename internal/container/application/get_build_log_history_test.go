package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

func TestGetBuildLogHistoryUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create completed build with all required fields
	pipelineRunName := "image-build-push-run-abc123"
	startedAt := time.Now().Add(-1 * time.Hour)
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		1,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-123"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		&finishedAt,
	)

	// Mock FindLatestByContainerID - returns completed build
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(completedBuild, nil)

	// Mock QueryPipelineRunLogsHTTP
	// Note: UseCase now adds 5-minute buffer to finishedAt
	expectedEndTime := finishedAt.Add(5 * time.Minute)
	mockLogData := io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, expectedEndTime, 1000).
		Return(mockLogData, nil)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, logData)

	// Verify we can read from the response
	data, _ := io.ReadAll(logData)
	assert.Contains(t, string(data), "success")

	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
}

func TestGetBuildLogHistoryUseCase_Execute_NoBuildHistory(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 999, // Non-existent container
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(999)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(999)).
		Return([]*build_history.BuildHistory{}, nil)

	// Mock FindLatestByContainerID - no build history
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(999)).
		Return(nil, projecterrors.ErrBuildHistoryNotFound)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logData)

	mockBuildHistoryRepo.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestGetBuildLogHistoryUseCase_Execute_BuildNotCompleted(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create running build (not completed)
	pipelineRunName := "image-build-push-run-running"
	startedAt := time.Now().Add(-10 * time.Minute)

	runningBuild, _ := build_history.ReconstructBuildHistory(
		2,
		10,
		build_history.BuildHistoryStatusRunning, // Not completed
		nil,
		stringPtr("event-456"),
		&pipelineRunName,
		nil,
		time.Now().Add(-20*time.Minute),
		&startedAt,
		nil, // No finishedAt - still running
	)

	// Mock FindLatestByContainerID - returns running build
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(runningBuild, nil)

	// Mock QueryPipelineRunLogsHTTP for running build
	// Note: Running builds use time.Now() + 5min buffer as endTime (capped at current time)
	// We can't predict exact time.Now(), so we verify it's called with correct pipeline name
	mockLogData := io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, mock.Anything, 1000).
		Return(mockLogData, nil)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert - should succeed for running builds (now supported)
	assert.NoError(t, err)
	assert.NotNil(t, logData)

	// Verify we can read from the response
	data, _ := io.ReadAll(logData)
	assert.Contains(t, string(data), "success")

	mockBuildHistoryRepo.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetBuildLogHistoryUseCase_Execute_NoPipelineRunName(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create completed build WITHOUT PipelineRunName
	startedAt := time.Now().Add(-1 * time.Hour)
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		3,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-789"),
		nil, // No PipelineRunName
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		&finishedAt,
	)

	// Mock FindLatestByContainerID - returns build without PipelineRunName
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(completedBuild, nil)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logData)

	mockBuildHistoryRepo.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestGetBuildLogHistoryUseCase_Execute_NoStartedAt(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create completed build WITHOUT StartedAt
	pipelineRunName := "image-build-push-run-no-start"
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		4,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-abc"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		nil, // No StartedAt
		&finishedAt,
	)

	// Mock FindLatestByContainerID
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(completedBuild, nil)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrBuildHistoryNotFound, err)
	assert.Nil(t, logData)

	mockBuildHistoryRepo.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
	// Loki client should not be called
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestGetBuildLogHistoryUseCase_Execute_NoFinishedAt(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create completed build WITHOUT FinishedAt (shouldn't happen in reality)
	pipelineRunName := "image-build-push-run-no-finish"
	startedAt := time.Now().Add(-1 * time.Hour)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		5,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-def"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		nil, // No FinishedAt
	)

	// Mock FindLatestByContainerID
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(completedBuild, nil)

	// Mock QueryPipelineRunLogsHTTP for build without finishedAt
	// Note: Build without finishedAt uses time.Now() + 5min buffer as endTime (capped at current time)
	// We can't predict exact time.Now(), so we verify it's called with correct pipeline name
	mockLogData := io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, mock.Anything, 1000).
		Return(mockLogData, nil)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert - should succeed (treat missing finishedAt like running build)
	assert.NoError(t, err)
	assert.NotNil(t, logData)

	// Verify we can read from the response
	data, _ := io.ReadAll(logData)
	assert.Contains(t, string(data), "success")

	mockBuildHistoryRepo.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetBuildLogHistoryUseCase_Execute_LokiQueryFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Mock FindActiveByContainerID - no active builds
	mockBuildHistoryRepo.On("FindActiveByContainerID", ctx, uint(10)).
		Return([]*build_history.BuildHistory{}, nil)

	// Create valid completed build
	pipelineRunName := "image-build-push-run-loki-fail"
	startedAt := time.Now().Add(-1 * time.Hour)
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		6,
		10,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-ghi"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		&finishedAt,
	)

	// Mock FindLatestByContainerID
	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, uint(10)).
		Return(completedBuild, nil)

	// Mock QueryPipelineRunLogsHTTP - fails
	// Note: UseCase now adds 5-minute buffer to finishedAt
	expectedEndTime := finishedAt.Add(5 * time.Minute)
	lokiError := errors.New("loki connection failed")
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, expectedEndTime, 1000).
		Return(nil, lokiError)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, lokiError, err)
	assert.Nil(t, logData)

	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
	mockPermissionService.AssertExpectations(t)
}

func TestGetBuildLogHistoryUseCase_Execute_PermissionDenied(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	log := logger.NewForTest()

	useCase := NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		log,
	)

	input := GetBuildLogHistoryInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user does NOT have access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).
		Return(containererrors.ErrPermissionDenied)

	// Execute
	logData, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrPermissionDenied, err)
	assert.Nil(t, logData)

	mockPermissionService.AssertExpectations(t)
	// BuildHistoryRepo and LokiClient should not be called when permission denied
	mockBuildHistoryRepo.AssertNotCalled(t, "FindLatestByContainerID")
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}
