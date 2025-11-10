package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projectservice "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestDeleteContainerUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockVolumeSvc := new(projectservice.MockVolumeService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteContainerUseCase(mockRepo, mockPermSvc, mockVolumeSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := DeleteContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("SoftDeleteNetworksByContainerID", ctx, containerID).Return(nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	// VolumeService should not be called when container has no mounts
	mockVolumeSvc.AssertNotCalled(t, "DeleteVolume")
}

func TestDeleteContainerUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockVolumeSvc := new(projectservice.MockVolumeService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteContainerUseCase(mockRepo, mockPermSvc, mockVolumeSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := DeleteContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(assert.AnError)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByIDForUpdate")
	mockRepo.AssertNotCalled(t, "Save")
}

func TestDeleteContainerUseCase_Execute_ContainerNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockVolumeSvc := new(projectservice.MockVolumeService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteContainerUseCase(mockRepo, mockPermSvc, mockVolumeSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := DeleteContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(nil, assert.AnError)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}
func TestDeleteContainerUseCase_Execute_WithVolumes_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockVolumeSvc := new(projectservice.MockVolumeService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteContainerUseCase(mockRepo, mockPermSvc, mockVolumeSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := DeleteContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	// Create container with 2 mounts
	mockContainer := createMockContainerWithMounts(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("SoftDeleteNetworksByContainerID", ctx, containerID).Return(nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	// Expect DeleteVolume to be called for each mount
	mockVolumeSvc.On("DeleteVolume", ctx, uint(1000)).Return(nil)
	mockVolumeSvc.On("DeleteVolume", ctx, uint(1001)).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockVolumeSvc.AssertExpectations(t)
	mockVolumeSvc.AssertNumberOfCalls(t, "DeleteVolume", 2)
}

func TestDeleteContainerUseCase_Execute_WithVolumes_PartialFailure(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockVolumeSvc := new(projectservice.MockVolumeService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteContainerUseCase(mockRepo, mockPermSvc, mockVolumeSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := DeleteContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	// Create container with 2 mounts
	mockContainer := createMockContainerWithMounts(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("SoftDeleteNetworksByContainerID", ctx, containerID).Return(nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	// First volume delete succeeds, second fails
	mockVolumeSvc.On("DeleteVolume", ctx, uint(1000)).Return(nil)
	mockVolumeSvc.On("DeleteVolume", ctx, uint(1001)).Return(assert.AnError)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	// Should still succeed even if some volumes fail to delete
	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockVolumeSvc.AssertExpectations(t)
	mockVolumeSvc.AssertNumberOfCalls(t, "DeleteVolume", 2)
}
