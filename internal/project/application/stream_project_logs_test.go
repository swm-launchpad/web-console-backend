package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

func TestStreamProjectLogsUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockKubeClient := new(infrastructure.MockKubeClient)
	log := logger.NewForTest()

	useCase := NewStreamProjectLogsUseCase(
		mockProjectRepo,
		mockLokiClient,
		mockKubeClient,
		log,
	)

	input := StreamProjectLogsInput{
		ProjectID: 10,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock CheckApplicationPodsRunning - pods are running
	mockKubeClient.On("CheckApplicationPodsRunning", ctx, "p2025011812000012345678").
		Return(true, nil)

	// Mock StreamApplicationLogs
	mockLogStream := io.NopCloser(strings.NewReader("log data"))
	mockLokiClient.On("StreamApplicationLogs", ctx, "p2025011812000012345678", mock.AnythingOfType("time.Time")).
		Return(mockLogStream, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, logStream)

	// Verify we can read from the stream
	data, _ := io.ReadAll(logStream)
	assert.Equal(t, "log data", string(data))

	mockProjectRepo.AssertExpectations(t)
	mockKubeClient.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestStreamProjectLogsUseCase_Execute_ProjectNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockKubeClient := new(infrastructure.MockKubeClient)
	log := logger.NewForTest()

	useCase := NewStreamProjectLogsUseCase(
		mockProjectRepo,
		mockLokiClient,
		mockKubeClient,
		log,
	)

	input := StreamProjectLogsInput{
		ProjectID: 999, // Non-existent project
	}

	// Mock FindByID - project not found
	mockProjectRepo.On("FindByID", ctx, uint(999)).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrProjectNotFound, err)
	assert.Nil(t, logStream)

	mockProjectRepo.AssertExpectations(t)
	// KubeClient and LokiClient should not be called
	mockKubeClient.AssertNotCalled(t, "CheckApplicationPodsRunning")
	mockLokiClient.AssertNotCalled(t, "StreamApplicationLogs")
}

func TestStreamProjectLogsUseCase_Execute_NoPodsRunning(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockKubeClient := new(infrastructure.MockKubeClient)
	log := logger.NewForTest()

	useCase := NewStreamProjectLogsUseCase(
		mockProjectRepo,
		mockLokiClient,
		mockKubeClient,
		log,
	)

	input := StreamProjectLogsInput{
		ProjectID: 10,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock CheckApplicationPodsRunning - NO pods running (scale-to-zero)
	mockKubeClient.On("CheckApplicationPodsRunning", ctx, "p2025011812000012345678").
		Return(false, nil)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.EqualError(t, err, "no application pods running")
	assert.Nil(t, logStream)

	mockProjectRepo.AssertExpectations(t)
	mockKubeClient.AssertExpectations(t)
	// LokiClient should not be called when no pods running
	mockLokiClient.AssertNotCalled(t, "StreamApplicationLogs")
}

func TestStreamProjectLogsUseCase_Execute_KubeCheckFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockKubeClient := new(infrastructure.MockKubeClient)
	log := logger.NewForTest()

	useCase := NewStreamProjectLogsUseCase(
		mockProjectRepo,
		mockLokiClient,
		mockKubeClient,
		log,
	)

	input := StreamProjectLogsInput{
		ProjectID: 10,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock CheckApplicationPodsRunning - kube API fails
	kubeError := errors.New("kubernetes API unavailable")
	mockKubeClient.On("CheckApplicationPodsRunning", ctx, "p2025011812000012345678").
		Return(false, kubeError)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, kubeError, err)
	assert.Nil(t, logStream)

	mockProjectRepo.AssertExpectations(t)
	mockKubeClient.AssertExpectations(t)
	// LokiClient should not be called when kube check fails
	mockLokiClient.AssertNotCalled(t, "StreamApplicationLogs")
}

func TestStreamProjectLogsUseCase_Execute_LokiStreamFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockKubeClient := new(infrastructure.MockKubeClient)
	log := logger.NewForTest()

	useCase := NewStreamProjectLogsUseCase(
		mockProjectRepo,
		mockLokiClient,
		mockKubeClient,
		log,
	)

	input := StreamProjectLogsInput{
		ProjectID: 10,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock CheckApplicationPodsRunning - pods are running
	mockKubeClient.On("CheckApplicationPodsRunning", ctx, "p2025011812000012345678").
		Return(true, nil)

	// Mock StreamApplicationLogs - Loki fails
	lokiError := errors.New("loki connection failed")
	mockLokiClient.On("StreamApplicationLogs", ctx, "p2025011812000012345678", mock.AnythingOfType("time.Time")).
		Return(nil, lokiError)

	// Execute
	logStream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, lokiError, err)
	assert.Nil(t, logStream)

	mockProjectRepo.AssertExpectations(t)
	mockKubeClient.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}
