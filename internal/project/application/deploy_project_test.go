package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestDeployProjectUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	testLogger := logger.NewForTest()
	useCase := NewDeployProjectUseCase(mockDeployService, testLogger)

	input := DeployProjectInput{
		ProjectID: 1,
	}

	mockDeployService.On("BuildAndDeployProject", mock.Anything, uint(1)).Return(nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "Build and deployment initiated", output.Message)
	assert.Equal(t, uint(1), output.ProjectID)

	mockDeployService.AssertExpectations(t)
}

// Note: Permission denial test removed - permission check moved to handler
// The handler is responsible for checking permissions and converting errors to ErrProjectNotFound

func TestDeployProjectUseCase_Execute_ProjectAlreadyDeploying(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	testLogger := logger.NewForTest()
	useCase := NewDeployProjectUseCase(mockDeployService, testLogger)

	input := DeployProjectInput{
		ProjectID: 1,
	}

	mockDeployService.On("BuildAndDeployProject", mock.Anything, uint(1)).
		Return(projecterrors.ErrProjectAlreadyDeploying)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrProjectAlreadyDeploying))

	mockDeployService.AssertExpectations(t)
}

func TestDeployProjectUseCase_Execute_ContainerConfigNotFound(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	testLogger := logger.NewForTest()
	useCase := NewDeployProjectUseCase(mockDeployService, testLogger)

	input := DeployProjectInput{
		ProjectID: 1,
	}

	mockDeployService.On("BuildAndDeployProject", mock.Anything, uint(1)).
		Return(projecterrors.ErrContainerConfigNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrContainerConfigNotFound))

	mockDeployService.AssertExpectations(t)
}
