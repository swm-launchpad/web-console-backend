package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestUpdateSecretUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateSecretUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"
	newValue := "new_super_secret_password"

	input := UpdateSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       newValue,
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
	assert.Equal(t, newValue, output.Value)
	assert.NotEmpty(t, output.UpdatedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateSecretUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateSecretUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := UpdateSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "DATABASE_PASSWORD",
		Value:       "new_value",
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

func TestUpdateSecretUseCase_Execute_SecretNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateSecretUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := UpdateSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "NONEXISTENT_SECRET",
		Value:       "new_value",
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

func TestUpdateSecretUseCase_Execute_ValueTooLong(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateSecretUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"
	value := make([]byte, 5001)
	for i := range value {
		value[i] = 'a'
	}

	input := UpdateSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       string(value),
	}

	mockContainer := createMockContainerWithSecretsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrSecretValueTooLong)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestUpdateSecretUseCase_Execute_EmptyValue(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateSecretUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := UpdateSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         "DATABASE_PASSWORD",
		Value:       "",
	}

	mockContainer := createMockContainerWithSecretsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrSecretValueRequired)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}
