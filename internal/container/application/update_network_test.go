package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestUpdateNetworkUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	networkID := uint(200)
	newPort := uint16(9090)
	sameFQDN := "oldapp.launchpad.kr" // Keep FQDN unchanged to test regular update path

	input := UpdateNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		NetworkID:    networkID,
		InternalPort: &newPort,
		NetworkType:  string(value.NetworkTypeHTTP),
		FQDN:         &sameFQDN,
	}

	mockContainer := createMockContainer(containerID, projectID)
	initialPort := uint16(8080)
	initialFQDN := "oldapp.launchpad.kr"
	externalPort := uint16(18080)
	network := createMockNetworkWithFQDN(networkID, containerID, &initialPort, &externalPort, value.NetworkTypeHTTP, &initialFQDN)
	_ = mockContainer.AddNetworkDirect(network)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("CheckInternalPortExistsInProjectExcludingSelf", ctx, projectID, newPort, networkID).Return(false, nil)
	mockRepo.On("CheckFQDNExistsForProjectExcludingSelf", ctx, sameFQDN, networkID, projectID).Return(false, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, networkID, output.NetworkID)
	assert.Equal(t, newPort, *output.InternalPort)
	assert.Equal(t, string(value.NetworkTypeHTTP), output.NetworkType)
	assert.Equal(t, sameFQDN, *output.FQDN)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_PartialUpdate(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	networkID := uint(200)
	newPort := uint16(9090)
	sameFQDN := "oldapp.launchpad.kr" // Keep FQDN unchanged

	input := UpdateNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		NetworkID:    networkID,
		InternalPort: &newPort, // Updating port
		NetworkType:  "",       // Not updating
		FQDN:         &sameFQDN,
	}

	mockContainer := createMockContainer(containerID, projectID)
	initialPort := uint16(8080)
	initialFQDN := "oldapp.launchpad.kr"
	externalPort := uint16(18080)
	network := createMockNetworkWithFQDN(networkID, containerID, &initialPort, &externalPort, value.NetworkTypeHTTP, &initialFQDN)
	_ = mockContainer.AddNetworkDirect(network)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("CheckInternalPortExistsInProjectExcludingSelf", ctx, projectID, newPort, networkID).Return(false, nil)
	mockRepo.On("CheckFQDNExistsForProjectExcludingSelf", ctx, sameFQDN, networkID, projectID).Return(false, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, newPort, *output.InternalPort) // Updated
	assert.Equal(t, sameFQDN, *output.FQDN)        // Unchanged

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := UpdateNetworkInput{
		ContainerID: containerID,
		UserID:      userID,
		NetworkID:   200,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).
		Return(containererrors.ErrPermissionDenied)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil) // Return nil so fn executes

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, containererrors.ErrPermissionDenied))

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_ContainerNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(999)
	userID := uint(100)

	input := UpdateNetworkInput{
		ContainerID: containerID,
		UserID:      userID,
		NetworkID:   200,
	}

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(nil, sql.ErrNoRows)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_DuplicateInternalPort(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	networkID := uint(200)
	duplicatePort := uint16(8080)

	input := UpdateNetworkInput{
		ContainerID:  containerID,
		UserID:       userID,
		NetworkID:    networkID,
		InternalPort: &duplicatePort,
	}

	mockContainer := createMockContainer(containerID, projectID)
	initialPort := uint16(9090)
	initialFQDN := "app.launchpad.kr"
	externalPort := uint16(19090)
	network := createMockNetworkWithFQDN(networkID, containerID, &initialPort, &externalPort, value.NetworkTypeHTTP, &initialFQDN)
	_ = mockContainer.AddNetworkDirect(network)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("CheckInternalPortExistsInProjectExcludingSelf", ctx, projectID, duplicatePort, networkID).Return(true, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, containererrors.ErrDuplicateInternalPort))

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_DuplicateFQDN(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	networkID := uint(200)
	duplicateFQDN := "existing.launchpad.kr"

	input := UpdateNetworkInput{
		ContainerID: containerID,
		UserID:      userID,
		NetworkID:   networkID,
		FQDN:        &duplicateFQDN,
	}

	mockContainer := createMockContainer(containerID, projectID)
	initialPort := uint16(8080)
	initialFQDN := "original.launchpad.kr"
	externalPort := uint16(18080)
	network := createMockNetworkWithFQDN(networkID, containerID, &initialPort, &externalPort, value.NetworkTypeHTTP, &initialFQDN)
	_ = mockContainer.AddNetworkDirect(network)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("CheckFQDNExistsForProjectExcludingSelf", ctx, duplicateFQDN, networkID, projectID).Return(true, nil)

	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).
		Return(nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, containererrors.ErrDuplicateFQDN))

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
}

func TestUpdateNetworkUseCase_Execute_NetworkNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewUpdateNetworkUseCase(mockRepo, mockPermSvc, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	nonExistentNetworkID := uint(999)

	input := UpdateNetworkInput{
		ContainerID: containerID,
		UserID:      userID,
		NetworkID:   nonExistentNetworkID,
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
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, containererrors.ErrNetworkNotFound))

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
}
