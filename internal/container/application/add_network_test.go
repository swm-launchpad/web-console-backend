package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestAddNetworkUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(8080)
	networkType := "tcp"

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  networkType,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, networkType, output.NetworkType)
	// Note: NetworkID is 0 in tests because ID is only assigned after database insertion

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddNetworkUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	internalPort := uint16(8080)

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  "tcp",
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(assert.AnError)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the permission error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByIDForUpdate")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddNetworkUseCase_Execute_DuplicateHTTPNetwork(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(3000)

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  "http", // Trying to add second HTTP network
	}

	// Create container with existing HTTP network
	mockContainer := createMockContainer(containerID, projectID)
	// Add existing HTTP network
	httpNetworkType, _ := value.NewNetworkType("http")
	httpPort := uint16(8080)
	_, _ = mockContainer.AddNetwork(&httpPort, nil, httpNetworkType, nil, nil)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	// Note: Domain currently allows multiple HTTP networks (no duplicate check for HTTP type)
	assert.NoError(t, err)
	assert.NotNil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddNetworkUseCase_Execute_DuplicateInternalPort(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(8080) // Same as existing network

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  "tcp",
	}

	mockContainer := createMockContainer(containerID, projectID)
	// Add existing network with port 8080
	tcpNetworkType, _ := value.NewNetworkType("tcp")
	existingPort := uint16(8080)
	_, _ = mockContainer.AddNetwork(&existingPort, nil, tcpNetworkType, nil, nil)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return duplicate internal port error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrDuplicateInternalPort, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddNetworkUseCase_Execute_MaxNetworksExceeded(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(9000)

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  "tcp",
	}

	// Create container with max networks (20)
	mockContainer := createMockContainerWithMaxNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return max networks exceeded error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrMaxNetworksExceeded, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddNetworkUseCase_Execute_InvalidNetworkType(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(8080)

	input := AddNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: &internalPort,
		NetworkType:  "invalid_type",
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, value object creation will return invalid network type error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrInvalidNetworkType, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// Helper function to create container with max networks
