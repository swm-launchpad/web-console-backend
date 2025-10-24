package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	userinfra "github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
)

func TestUpdateContainerUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	buildChangeDetector := service.NewBuildChangeDetector()
	mockTxMgr := new(db.MockTxManager)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	testLogger := logger.NewForTest()
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, buildChangeDetector, mockInstallationRepo, mockTxMgr, testLogger)

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
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	testLogger := logger.NewForTest()
	buildChangeDetector := service.NewBuildChangeDetector()
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, buildChangeDetector, mockInstallationRepo, mockTxMgr, testLogger)

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

func TestUpdateContainerUseCase_Execute_UnsetGitHubInstallationID(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockTxMgr := new(db.MockTxManager)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	testLogger := logger.NewForTest()
	buildChangeDetector := service.NewBuildChangeDetector()
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, buildChangeDetector, mockInstallationRepo, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	// Request to unset GitHubInstallationID (set to nil)
	input := UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userID,
		GitHubInstallationID:       nil,
		UpdateGitHubInstallationID: true, // Flag indicates we want to update it
	}

	mockContainer := createMockContainer(containerID, projectID)
	// Container initially has a GitHub installation ID
	installationID := int64(12345)
	mockContainer.SetGitHubInstallationID(&installationID)
	assert.NotNil(t, mockContainer.GitHubInstallationID())

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	// Verify that GitHubInstallationID was set to nil
	assert.Nil(t, mockContainer.GitHubInstallationID())

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateContainerUseCase_Execute_SetGitHubInstallationID(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockTxMgr := new(db.MockTxManager)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	testLogger := logger.NewForTest()
	buildChangeDetector := service.NewBuildChangeDetector()
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, buildChangeDetector, mockInstallationRepo, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	newInstallationID := int64(67890)

	// Request to set GitHubInstallationID to a specific value
	input := UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userID,
		GitHubInstallationID:       &newInstallationID,
		UpdateGitHubInstallationID: true,
	}

	mockContainer := createMockContainer(containerID, projectID)
	// Container initially has no GitHub installation ID
	assert.Nil(t, mockContainer.GitHubInstallationID())

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockInstallationRepo.On("ValidateUserOwnership", mock.Anything, newInstallationID, userID).Return(nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	// Verify that GitHubInstallationID was set to the new value
	assert.NotNil(t, mockContainer.GitHubInstallationID())
	assert.Equal(t, newInstallationID, *mockContainer.GitHubInstallationID())

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestUpdateContainerUseCase_Execute_NoUpdateGitHubInstallationID(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockTxMgr := new(db.MockTxManager)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	testLogger := logger.NewForTest()
	buildChangeDetector := service.NewBuildChangeDetector()
	useCase := NewUpdateContainerUseCase(mockRepo, mockPermSvc, mockResourceValidationSvc, buildChangeDetector, mockInstallationRepo, mockTxMgr, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)
	newName := "updated-name"

	// Request to update name but NOT update GitHubInstallationID
	input := UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userID,
		Name:                       &newName,
		GitHubInstallationID:       nil,
		UpdateGitHubInstallationID: false, // Flag is false - don't update
	}

	mockContainer := createMockContainer(containerID, projectID)
	// Container initially has a GitHub installation ID
	originalInstallationID := int64(12345)
	mockContainer.SetGitHubInstallationID(&originalInstallationID)

	mockPermSvc.On("CanUserModifyContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByIDForUpdate", ctx, containerID).Return(mockContainer, nil)
	mockRepo.On("Save", ctx, mockContainer).Return(nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	// Verify that GitHubInstallationID was NOT changed
	assert.NotNil(t, mockContainer.GitHubInstallationID())
	assert.Equal(t, originalInstallationID, *mockContainer.GitHubInstallationID())

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
