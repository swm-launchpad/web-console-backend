package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure"
	userinfra "github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
)

func TestCreateContainerUseCase_Execute_Success(t *testing.T) {
	mockContainerService := new(infrastructure.MockContainerService)
	mockContainerRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockVolumeService := new(projectinfra.MockVolumeService)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewCreateContainerUseCase(mockContainerService, mockContainerRepo, mockPermSvc, mockResourceValidationSvc, mockVolumeService, mockInstallationRepo, mockTxMgr, testLogger)

	ctx := context.Background()
	projectID := uint(10)
	userID := uint(100)
	containerID := uint(1)

	input := CreateContainerInput{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        "test-container",
		GitURL:      "https://github.com/test/repo",
		GitBranch:   "main",
		CPULimit:    1000,
		MemoryLimit: 2048,
	}

	mockContainer := createMockContainer(containerID, projectID)

	mockPermSvc.On("CanUserCreateContainer", ctx, userID, projectID).Return(nil)
	mockResourceValidationSvc.On("ValidateProjectResourceLimits", ctx, projectID, uint32(1000), uint32(2048), uint(0)).Return(nil)
	mockContainerService.On("CreateContainer", ctx, projectID, input.Name, mock.Anything, mock.Anything, (*uint)(nil), map[string]interface{}(nil), (*int64)(nil)).Return(mockContainer, nil)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, projectID, output.ProjectID)
	assert.Equal(t, "Test Container", output.Name)
	assert.NotEmpty(t, output.Slug)
	assert.NotEmpty(t, output.CreatedAt)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockContainerService.AssertExpectations(t)
}

func TestCreateContainerUseCase_Execute_PermissionDenied(t *testing.T) {
	mockContainerService := new(infrastructure.MockContainerService)
	mockContainerRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	mockResourceValidationSvc := new(infrastructure.MockResourceValidationService)
	mockVolumeService := new(projectinfra.MockVolumeService)
	mockInstallationRepo := new(userinfra.MockGitHubInstallationRepository)
	mockTxMgr := new(db.MockTxManager)
	testLogger := logger.NewForTest()
	useCase := NewCreateContainerUseCase(mockContainerService, mockContainerRepo, mockPermSvc, mockResourceValidationSvc, mockVolumeService, mockInstallationRepo, mockTxMgr, testLogger)

	ctx := context.Background()
	projectID := uint(10)
	userID := uint(100)

	input := CreateContainerInput{
		ProjectID:   projectID,
		UserID:      userID,
		Name:        "test-container",
		GitURL:      "https://github.com/test/repo",
		GitBranch:   "main",
		CPULimit:    1000,
		MemoryLimit: 2048,
	}

	mockPermSvc.On("CanUserCreateContainer", ctx, userID, projectID).Return(assert.AnError)
	mockTxMgr.On("RunInTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockTxMgr.AssertExpectations(t)
	mockPermSvc.AssertExpectations(t)
	mockContainerService.AssertNotCalled(t, "CreateContainer")
}
