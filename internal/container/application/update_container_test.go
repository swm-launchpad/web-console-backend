package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestUpdateContainerUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	newName := "updated-container"

	input := UpdateContainerInput{
		ContainerID: containerID,
		UserID:      userID,
		Name:        &newName,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.NotEmpty(t, output.UpdatedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateContainerUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	newName := "updated-container"

	input := UpdateContainerInput{
		ContainerID: containerID,
		UserID:      userID,
		Name:        &newName,
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
