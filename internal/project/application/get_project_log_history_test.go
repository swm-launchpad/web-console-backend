package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

func TestGetProjectLogHistoryUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	before := time.Now()
	input := GetProjectLogHistoryInput{
		ProjectID: 10,
		Before:    before,
		Limit:     100,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock QueryApplicationLogsHistory
	mockLogs := []infrastructure.ApplicationLogEntry{
		{
			Timestamp:     time.Now().Add(-1 * time.Hour),
			ContainerName: "app",
			LogLine:       "Application started",
		},
		{
			Timestamp:     time.Now().Add(-2 * time.Hour),
			ContainerName: "app",
			LogLine:       "Connecting to database",
		},
	}
	mockLokiClient.On("QueryApplicationLogsHistory", ctx, "p2025011812000012345678", before, 100).
		Return(mockLogs, nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Logs, 2)
	assert.Equal(t, "Application started", output.Logs[0].LogLine)
	assert.Equal(t, "Connecting to database", output.Logs[1].LogLine)

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetProjectLogHistoryUseCase_Execute_DefaultLimit(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	// Input with zero limit (should use default 100)
	input := GetProjectLogHistoryInput{
		ProjectID: 10,
		Before:    time.Time{}, // Zero value
		Limit:     0,           // Zero value - should default to 100
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock QueryApplicationLogsHistory - should be called with limit=100
	mockLogs := []infrastructure.ApplicationLogEntry{}
	mockLokiClient.On("QueryApplicationLogsHistory", ctx, "p2025011812000012345678", mock.AnythingOfType("time.Time"), 100).
		Return(mockLogs, nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Logs, 0)

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetProjectLogHistoryUseCase_Execute_ProjectNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	input := GetProjectLogHistoryInput{
		ProjectID: 999, // Non-existent project
		Before:    time.Now(),
		Limit:     100,
	}

	// Mock FindByID - project not found
	mockProjectRepo.On("FindByID", ctx, uint(999)).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrProjectNotFound, err)
	assert.Nil(t, output)

	mockProjectRepo.AssertExpectations(t)
	// LokiClient should not be called
	mockLokiClient.AssertNotCalled(t, "QueryApplicationLogsHistory")
}

func TestGetProjectLogHistoryUseCase_Execute_LokiQueryFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	before := time.Now()
	input := GetProjectLogHistoryInput{
		ProjectID: 10,
		Before:    before,
		Limit:     100,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock QueryApplicationLogsHistory - Loki fails
	lokiError := errors.New("loki query timeout")
	mockLokiClient.On("QueryApplicationLogsHistory", ctx, "p2025011812000012345678", before, 100).
		Return(nil, lokiError)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, lokiError, err)
	assert.Nil(t, output)

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetProjectLogHistoryUseCase_Execute_EmptyLogs(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	before := time.Now()
	input := GetProjectLogHistoryInput{
		ProjectID: 10,
		Before:    before,
		Limit:     100,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock QueryApplicationLogsHistory - returns empty logs
	mockLogs := []infrastructure.ApplicationLogEntry{}
	mockLokiClient.On("QueryApplicationLogsHistory", ctx, "p2025011812000012345678", before, 100).
		Return(mockLogs, nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert - empty logs is still success
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Logs, 0)

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestGetProjectLogHistoryUseCase_Execute_CustomLimit(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	log := logger.NewForTest()

	useCase := NewGetProjectLogHistoryUseCase(
		mockProjectRepo,
		mockLokiClient,
		log,
	)

	before := time.Now()
	input := GetProjectLogHistoryInput{
		ProjectID: 10,
		Before:    before,
		Limit:     50, // Custom limit
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Mock QueryApplicationLogsHistory - should be called with limit=50
	mockLogs := make([]infrastructure.ApplicationLogEntry, 50)
	for i := 0; i < 50; i++ {
		mockLogs[i] = infrastructure.ApplicationLogEntry{
			Timestamp:     time.Now().Add(-time.Duration(i) * time.Minute),
			ContainerName: "app",
			LogLine:       "log line",
		}
	}
	mockLokiClient.On("QueryApplicationLogsHistory", ctx, "p2025011812000012345678", before, 50).
		Return(mockLogs, nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Logs, 50)

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}
