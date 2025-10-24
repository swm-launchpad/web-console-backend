package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestDeleteSecretUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"

	input := DeleteSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
	}

	mockContainer := createMockContainerWithSecretsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, key, output.Key)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDeleteSecretUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := DeleteSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "DATABASE_PASSWORD",
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

func TestDeleteSecretUseCase_Execute_SecretNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := DeleteSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "NONEXISTENT_SECRET",
	}

	mockContainer := createMockContainerWithSecretsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrSecretNotFound)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestDeleteSecretUseCase_Execute_ContainerNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewDeleteSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := DeleteSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "DATABASE_PASSWORD",
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
