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

func TestAddSecretUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"
	value := "super_secret_password_123"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, key, output.Key)
	assert.Equal(t, value, output.Value)
	// Note: SecretID is 0 in tests because ID is only assigned after database insertion
	assert.NotEmpty(t, output.CreatedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestAddSecretUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	key := "API_SECRET"
	value := "secret_value"

	input := AddSecretInput{
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

func TestAddSecretUseCase_Execute_DuplicateKey(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"
	value := "secret_value"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	// Create container with an existing secret with the same key
	mockContainer := createMockContainerWithSecretsAndNetworks(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the duplicate error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrDuplicateSecretKey)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_MaxSecretsExceeded(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "EXTRA_SECRET"
	value := "value"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	// Create container with 100 secrets (maximum)
	mockContainer := createMockContainerWithMaxSecrets(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the max exceeded error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrMaxSecretsExceeded)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_InvalidKeyFormat(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "invalid-key" // hyphens not allowed
	value := "secret_value"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the validation error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrInvalidSecretKey)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_KeyTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	// Create a 256-character key (all 'A's) - exceeds 255 limit
	key := ""
	for i := 0; i < 256; i++ {
		key += "A"
	}
	value := "secret_value"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the validation error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrSecretKeyTooLong)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_ValueTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "DATABASE_PASSWORD"
	// Create a 5001-character value - exceeds 5000 limit
	value := make([]byte, 5001)
	for i := range value {
		value[i] = 'a'
	}

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       string(value),
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the validation error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrSecretValueTooLong)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_EmptyKey(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := ""
	value := "secret_value"

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the validation error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrSecretKeyRequired)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}

func TestAddSecretUseCase_Execute_EmptyValue(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewAddSecretUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	key := "API_KEY"
	value := ""

	input := AddSecretInput{
		ContainerID: containerID,
		UserID:      userID,
		Key:         key,
		Value:       value,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Execute fn, which will return the validation error

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.ErrorIs(t, err, containererrors.ErrSecretValueRequired)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Save")
}
