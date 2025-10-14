package application

import (
	"context"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestDeleteEnvVarUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewDeleteEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL"

	input := DeleteEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
	}

	// Create container with existing env vars (including DATABASE_URL)
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
	assert.Equal(t, key, output.Key)
	assert.NotEmpty(t, output.DeletedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestDeleteEnvVarUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewDeleteEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	key := "API_KEY"

	input := DeleteEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
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

func TestDeleteEnvVarUseCase_Execute_EnvVarNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewDeleteEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "NON_EXISTENT_KEY"

	input := DeleteEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return env var not found error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrEnvVarNotFound, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestDeleteEnvVarUseCase_Execute_ContainerNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewDeleteEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(999) // Non-existent container
	userID := uint(100)
	key := "DATABASE_URL"

	input := DeleteEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(nil, containererrors.ErrContainerNotFound)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the repository error

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

func TestDeleteEnvVarUseCase_Execute_CannotDeleteFromDeleted(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	useCase := NewDeleteEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL"

	input := DeleteEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
	}

	// Create deleted container with env vars
	mockContainer := createMockDeletedContainer(containerID, projectID)
	// Add env var to deleted container for testing
	_ = mockContainer.AddEnvVarDirect(createMockEnvVar(1, containerID, "DATABASE_URL", "postgres://..."))

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic returns nil for deleted containers (silent success)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	// Note: Domain logic silently succeeds when deleting from a deleted container
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, key, output.Key)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
