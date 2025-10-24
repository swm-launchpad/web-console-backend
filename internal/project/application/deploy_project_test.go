package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
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

	// Create mock deployment
	d := deployment.NewDeployment(1)
	d.SetDeploymentID(100)
	eventID := "test-event-123"
	_ = d.InitTektonInfo(&eventID, nil)

	mockDeployService.On("DeployProject", mock.Anything, uint(1)).Return(d, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint64(100), output.DeploymentID)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "untracked", output.Status)
	assert.Equal(t, "test-event-123", output.TektonEventID)

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

	mockDeployService.On("DeployProject", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrProjectAlreadyDeploying)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrProjectAlreadyDeploying))

	mockDeployService.AssertExpectations(t)
}

func TestDeployProjectUseCase_Execute_WithOptionalFields(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	testLogger := logger.NewForTest()
	useCase := NewDeployProjectUseCase(mockDeployService, testLogger)

	input := DeployProjectInput{
		ProjectID: 1,
	}

	// Create mock deployment with all optional fields
	d := deployment.NewDeployment(1)
	d.SetDeploymentID(100)
	eventID := "test-event-123"
	runName := "test-run-123"
	_ = d.InitTektonInfo(&eventID, &runName)

	summary := "Deployment in progress"
	startedAt := time.Now()
	_ = d.UpdateRunningStatus(&summary, &startedAt)

	mockDeployService.On("DeployProject", mock.Anything, uint(1)).Return(d, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint64(100), output.DeploymentID)
	assert.Equal(t, "test-event-123", output.TektonEventID)
	assert.Equal(t, "test-run-123", output.TektonPipelineRunName)
	assert.Equal(t, "Deployment in progress", output.Summary)
	assert.NotEmpty(t, output.StartedAt)

	mockDeployService.AssertExpectations(t)
}
