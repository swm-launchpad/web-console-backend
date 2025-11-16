package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestGetContainerUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	testLogger := logger.NewForTest()
	useCase := NewGetContainerUseCase(mockRepo, mockTemplateRepo, mockPermSvc, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	// Create mock container with networks and env vars
	mockContainer := createMockContainerWithEnvVarsAndNetworks(containerID, projectID)

	input := GetContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	// Mock expectations
	mockPermSvc.On("CanUserAccessContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByID", ctx, containerID).Return(mockContainer, nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, containerID, output.ContainerID)
	assert.Equal(t, projectID, output.ProjectID)
	assert.Equal(t, "Test Container", output.Name)
	assert.Equal(t, "c2025011812000012345678", output.Slug)
	assert.Len(t, output.Networks, 2)
	assert.Len(t, output.EnvVars, 2)

	// Verify network fields
	assert.Equal(t, uint(1), output.Networks[0].NetworkID)
	assert.Equal(t, "tcp", output.Networks[0].NetworkType)
	assert.Equal(t, uint16(8080), output.Networks[0].InternalPort)

	// Verify env var fields
	assert.Equal(t, uint(1), output.EnvVars[0].EnvVarID)
	assert.Equal(t, "DATABASE_URL", output.EnvVars[0].Key)
	assert.Equal(t, "postgres://...", output.EnvVars[0].Value)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetContainerUseCase_Execute_PermissionDenied(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	testLogger := logger.NewForTest()
	useCase := NewGetContainerUseCase(mockRepo, mockTemplateRepo, mockPermSvc, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)

	input := GetContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	// Create mock container (use case calls FindByID before permission check)
	mockContainer := createMockContainer(containerID, 10)

	// Mock expectations - permission denied
	mockRepo.On("FindByID", ctx, containerID).Return(mockContainer, nil)
	mockPermSvc.On("CanUserAccessContainer", ctx, userID, containerID).Return(assert.AnError)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetContainerUseCase_Execute_ContainerNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	testLogger := logger.NewForTest()
	useCase := NewGetContainerUseCase(mockRepo, mockTemplateRepo, mockPermSvc, testLogger)

	ctx := context.Background()
	containerID := uint(999)
	userID := uint(100)

	input := GetContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	// Mock expectations - FindByID fails, so permission check is never called
	mockRepo.On("FindByID", ctx, containerID).Return(nil, assert.AnError)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)

	mockRepo.AssertExpectations(t)
	mockPermSvc.AssertNotCalled(t, "CanUserAccessContainer")
}

func TestGetContainerUseCase_Execute_WithOptionalFields(t *testing.T) {
	// Arrange
	mockRepo := new(infrastructure.MockContainerRepository)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	testLogger := logger.NewForTest()
	useCase := NewGetContainerUseCase(mockRepo, mockTemplateRepo, mockPermSvc, testLogger)

	ctx := context.Background()
	containerID := uint(1)
	userID := uint(100)
	projectID := uint(10)

	// Create container with optional fields
	fqdn := "myapp.launchpad.io"
	gitDir := "backend" // No leading slash (git path validation)
	stableWindow := uint32(300)
	templateConfig := map[string]interface{}{"framework": "react"}

	mockContainer := createMockContainerWithOptionalFields(
		containerID,
		projectID,
		nil, // fqdn moved to network
		&gitDir,
		&stableWindow,
		templateConfig,
	)

	// Add network with FQDN
	internalPort := uint16(8080)
	networkType, _ := value.NewNetworkType("http")
	_, _ = mockContainer.AddNetwork(&internalPort, nil, networkType, nil, &fqdn)

	input := GetContainerInput{
		ContainerID: containerID,
		UserID:      userID,
	}

	mockPermSvc.On("CanUserAccessContainer", ctx, userID, containerID).Return(nil)
	mockRepo.On("FindByID", ctx, containerID).Return(mockContainer, nil)

	// Act
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	require.Len(t, output.Networks, 1)
	assert.Equal(t, fqdn, output.Networks[0].FQDN)
	assert.Equal(t, gitDir, output.GitSubpath)
	assert.Equal(t, stableWindow, output.StableWindow)
	assert.Equal(t, templateConfig, output.TemplateConfig)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
