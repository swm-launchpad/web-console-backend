package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// MockKubeClient is a mock implementation of KubeClient interface for testing
type MockKubeClient struct {
	mock.Mock
}

func (m *MockKubeClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
	args := m.Called(ctx, pipelineRunName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PipelineRun), args.Error(1)
}

func (m *MockKubeClient) GetPipelineRunLogs(ctx context.Context, pipelineRunName string) (string, error) {
	args := m.Called(ctx, pipelineRunName)
	return args.String(0), args.Error(1)
}

func (m *MockKubeClient) ListPipelineRuns(ctx context.Context, projectID uint) ([]*dto.PipelineRun, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.PipelineRun), args.Error(1)
}

func (m *MockKubeClient) FindPipelineRunNameByEventID(ctx context.Context, eventID string) (string, error) {
	args := m.Called(ctx, eventID)
	return args.String(0), args.Error(1)
}

func (m *MockKubeClient) CheckApplicationPodsRunning(ctx context.Context, projectSlug string) (bool, error) {
	args := m.Called(ctx, projectSlug)
	return args.Bool(0), args.Error(1)
}

func (m *MockKubeClient) GetProjectPodStatus(ctx context.Context, projectID uint) (*dto.PodStatus, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PodStatus), args.Error(1)
}

func TestCheckProjectPodStatusUseCase_Execute_Success_PodExists(t *testing.T) {
	// Arrange
	mockKubeClient := new(MockKubeClient)
	testLogger := logger.NewForTest()

	useCase := NewCheckProjectPodStatusUseCase(mockKubeClient, testLogger)

	ctx := context.Background()
	projectID := uint(123)

	expectedPodStatus := &dto.PodStatus{
		Exists:          true,
		Phase:           "Running",
		ReadyContainers: 2,
		TotalContainers: 2,
	}

	// Set up mock expectations
	mockKubeClient.On("GetProjectPodStatus", ctx, projectID).Return(expectedPodStatus, nil)

	input := CheckProjectPodStatusInput{ProjectID: projectID}

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.Exists)
	assert.Equal(t, "Running", output.Phase)
	assert.Equal(t, 2, output.ReadyContainers)
	assert.Equal(t, 2, output.TotalContainers)

	mockKubeClient.AssertExpectations(t)
}

func TestCheckProjectPodStatusUseCase_Execute_Success_PodNotExists(t *testing.T) {
	// Arrange
	mockKubeClient := new(MockKubeClient)
	testLogger := logger.NewForTest()

	useCase := NewCheckProjectPodStatusUseCase(mockKubeClient, testLogger)

	ctx := context.Background()
	projectID := uint(456)

	expectedPodStatus := &dto.PodStatus{
		Exists:          false,
		Phase:           "",
		ReadyContainers: 0,
		TotalContainers: 0,
	}

	// Set up mock expectations
	mockKubeClient.On("GetProjectPodStatus", ctx, projectID).Return(expectedPodStatus, nil)

	input := CheckProjectPodStatusInput{ProjectID: projectID}

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.False(t, output.Exists)
	assert.Equal(t, "", output.Phase)
	assert.Equal(t, 0, output.ReadyContainers)
	assert.Equal(t, 0, output.TotalContainers)

	mockKubeClient.AssertExpectations(t)
}

func TestCheckProjectPodStatusUseCase_Execute_Success_PodPending(t *testing.T) {
	// Arrange
	mockKubeClient := new(MockKubeClient)
	testLogger := logger.NewForTest()

	useCase := NewCheckProjectPodStatusUseCase(mockKubeClient, testLogger)

	ctx := context.Background()
	projectID := uint(789)

	expectedPodStatus := &dto.PodStatus{
		Exists:          true,
		Phase:           "Pending",
		ReadyContainers: 0,
		TotalContainers: 2,
	}

	// Set up mock expectations
	mockKubeClient.On("GetProjectPodStatus", ctx, projectID).Return(expectedPodStatus, nil)

	input := CheckProjectPodStatusInput{ProjectID: projectID}

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.Exists)
	assert.Equal(t, "Pending", output.Phase)
	assert.Equal(t, 0, output.ReadyContainers)
	assert.Equal(t, 2, output.TotalContainers)

	mockKubeClient.AssertExpectations(t)
}

func TestCheckProjectPodStatusUseCase_Execute_KubernetesError(t *testing.T) {
	// Arrange
	mockKubeClient := new(MockKubeClient)
	testLogger := logger.NewForTest()

	useCase := NewCheckProjectPodStatusUseCase(mockKubeClient, testLogger)

	ctx := context.Background()
	projectID := uint(999)

	// Set up mock expectations
	mockKubeClient.On("GetProjectPodStatus", ctx, projectID).Return(nil, projecterrors.ErrKubernetesUnavailable)

	input := CheckProjectPodStatusInput{ProjectID: projectID}

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrKubernetesUnavailable, err)
	assert.Nil(t, output)

	mockKubeClient.AssertExpectations(t)
}

func TestCheckProjectPodStatusUseCase_Execute_Success_PartiallyReadyContainers(t *testing.T) {
	// Arrange
	mockKubeClient := new(MockKubeClient)
	testLogger := logger.NewForTest()

	useCase := NewCheckProjectPodStatusUseCase(mockKubeClient, testLogger)

	ctx := context.Background()
	projectID := uint(321)

	expectedPodStatus := &dto.PodStatus{
		Exists:          true,
		Phase:           "Running",
		ReadyContainers: 1,
		TotalContainers: 3,
	}

	// Set up mock expectations
	mockKubeClient.On("GetProjectPodStatus", ctx, projectID).Return(expectedPodStatus, nil)

	input := CheckProjectPodStatusInput{ProjectID: projectID}

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.Exists)
	assert.Equal(t, "Running", output.Phase)
	assert.Equal(t, 1, output.ReadyContainers)
	assert.Equal(t, 3, output.TotalContainers)

	mockKubeClient.AssertExpectations(t)
}
