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
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// mockReadCloser is a test helper that implements io.ReadCloser
type mockReadCloser struct {
	*strings.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(data string) io.ReadCloser {
	return &mockReadCloser{Reader: strings.NewReader(data)}
}

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

	// Mock QueryApplicationLogsHistoryRaw - returns raw stream
	mockStream := newMockReadCloser(`{"status":"success","data":{"result":[]}}`)
	mockLokiClient.On("QueryApplicationLogsHistoryRaw", ctx, "p2025011812000012345678", before, 100).
		Return(mockStream, nil)

	// Execute
	stream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	// Verify stream can be closed
	assert.NoError(t, stream.Close())

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

	// Mock QueryApplicationLogsHistoryRaw - should be called with limit=100
	mockStream := newMockReadCloser(`{"status":"success","data":{"result":[]}}`)
	mockLokiClient.On("QueryApplicationLogsHistoryRaw", ctx, "p2025011812000012345678", mock.AnythingOfType("time.Time"), 100).
		Return(mockStream, nil)

	// Execute
	stream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.NoError(t, stream.Close())

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
	stream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrProjectNotFound, err)
	assert.Nil(t, stream)

	mockProjectRepo.AssertExpectations(t)
	// LokiClient should not be called
	mockLokiClient.AssertNotCalled(t, "QueryApplicationLogsHistoryRaw")
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

	// Mock QueryApplicationLogsHistoryRaw - Loki fails
	lokiError := errors.New("loki query timeout")
	mockLokiClient.On("QueryApplicationLogsHistoryRaw", ctx, "p2025011812000012345678", before, 100).
		Return(nil, lokiError)

	// Execute
	stream, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, lokiError, err)
	assert.Nil(t, stream)

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

	// Mock QueryApplicationLogsHistoryRaw - returns empty result
	mockStream := newMockReadCloser(`{"status":"success","data":{"result":[]}}`)
	mockLokiClient.On("QueryApplicationLogsHistoryRaw", ctx, "p2025011812000012345678", before, 100).
		Return(mockStream, nil)

	// Execute
	stream, err := useCase.Execute(ctx, input)

	// Assert - empty logs is still success
	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.NoError(t, stream.Close())

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

	// Mock QueryApplicationLogsHistoryRaw - should be called with limit=50
	mockStream := newMockReadCloser(`{"status":"success","data":{"result":[]}}`)
	mockLokiClient.On("QueryApplicationLogsHistoryRaw", ctx, "p2025011812000012345678", before, 50).
		Return(mockStream, nil)

	// Execute
	stream, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.NoError(t, stream.Close())

	mockProjectRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}
