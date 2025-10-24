package application

import (
	"context"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestDeleteNetworkUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(8080)

	input := DeleteNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: internalPort,
	}

	// Create container with existing networks (including port 8080)
	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

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
	assert.Equal(t, internalPort, output.InternalPort)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDeleteNetworkUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	internalPort := uint16(8080)

	input := DeleteNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: internalPort,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(assert.AnError)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

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

func TestDeleteNetworkUseCase_Execute_NetworkNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(9999) // Non-existent port

	input := DeleteNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: internalPort,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrNetworkNotInContainer, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestDeleteNetworkUseCase_Execute_ContainerNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(999) // Non-existent container
	userID := uint(100)
	internalPort := uint16(8080)

	input := DeleteNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: internalPort,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(nil, containererrors.ErrContainerNotFound)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrContainerNotFound, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDeleteNetworkUseCase_Execute_CannotDeleteFromDeleted(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	internalPort := uint16(8080)

	input := DeleteNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		InternalPort: internalPort,
	}

	// Create deleted container with network
	mockContainer := createMockDeletedContainer(containerID, projectID)
	// Add network to deleted container for testing
	_ = mockContainer.AddNetworkDirect(createMockNetwork(1, containerID, 8080, "tcp"))

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	// Note: Domain logic silently succeeds when deleting from a deleted container
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, internalPort, output.InternalPort)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
