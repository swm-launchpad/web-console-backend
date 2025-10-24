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

func TestAddEnvVarUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL"
	value := "postgresql://localhost:5432/mydb"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	// Mock transaction manager to execute the function immediately
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, key, output.Key)
	assert.Equal(t, value, output.Value)
	// Note: EnvVarID is 0 in tests because ID is only assigned after database insertion
	assert.NotEmpty(t, output.CreatedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddEnvVarUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	key := "API_KEY"
	value := "secret"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
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

func TestAddEnvVarUseCase_Execute_DuplicateKey(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_URL" // Same as existing env var

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       "postgresql://other:5432/db",
	}

	// Create container with existing env var with same key
	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return duplicate error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrDuplicateEnvVarKey, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddEnvVarUseCase_Execute_MaxEnvVarsExceeded(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "NEW_VAR"
	value := "new_value"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	// Create container with max env vars (50)
	mockContainer := createMockContainerWithMaxEnvVars(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return max exceeded error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrMaxEnvVarsExceeded, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddEnvVarUseCase_Execute_InvalidKeyFormat(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "invalid-key-format" // Invalid: contains hyphens (only uppercase, numbers, underscores allowed)
	value := "value"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return invalid key error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrInvalidEnvVarKey, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddEnvVarUseCase_Execute_KeyTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	// Key longer than 255 characters
	key := "VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY_VERY_LONG_KEY"
	value := "value"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return key too long error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrEnvVarKeyTooLong, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddEnvVarUseCase_Execute_ValueTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "VALID_KEY"
	// Value longer than 5000 characters (max is 5000)
	value := make([]byte, 5001)
	for i := range value {
		value[i] = 'a'
	}

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       string(value),
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return value too long error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrEnvVarValueTooLong, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddEnvVarUseCase_Execute_EmptyKey(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "" // Empty key
	value := "value"

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return key required error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrEnvVarKeyRequired, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddEnvVarUseCase_Execute_EmptyValue(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddEnvVarUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "VALID_KEY"
	value := "" // Empty value

	input := AddEnvVarInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, domain logic will return value required error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrEnvVarValueRequired, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
