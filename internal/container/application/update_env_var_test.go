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

func TestUpdateEnvVarUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL"
	newValue := "postgresql://newhost:5432/newdb"

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   key,
		Value:       newValue,
	}

	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

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

func TestUpdateEnvVarUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   "DATABASE_URL",
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

func TestUpdateEnvVarUseCase_Execute_EnvVarNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   "NONEXISTENT_VAR",
		Value:       "new_value",
	}

	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrEnvVarNotFound)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestUpdateEnvVarUseCase_Execute_ValueTooLong(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL"
	value := make([]byte, 5001)
	for i := range value {
		value[i] = 'a'
	}

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   key,
		Value:       string(value),
	}

	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrEnvVarValueTooLong)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestUpdateEnvVarUseCase_Execute_EmptyValue(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   "DATABASE_URL",
		Value:       "",
	}

	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	// Empty value is not allowed for env vars
	output, err := useCase.Execute(ctx, input)

	assert.ErrorIs(t, err, containererrors.ErrEnvVarValueRequired)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestUpdateEnvVarUseCase_Execute_ContainerNotFound(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewUpdateEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := UpdateEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		EnvVarKey:   "DATABASE_URL",
		Value:       "new_value",
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
