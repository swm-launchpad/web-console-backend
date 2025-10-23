package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestRefreshDeploymentUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	useCase := NewRefreshDeploymentUseCase(mockDeployService)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Create refreshed deployment with updated status
	refreshedDeployment := deployment.NewDeployment(1)
	refreshedDeployment.SetDeploymentID(100)
	eventID := "test-event-123"
	runName := "test-run-123"
	_ = refreshedDeployment.InitTektonInfo(&eventID, &runName)
	summary := "Deployment running"
	startedAt := time.Now()
	_ = refreshedDeployment.UpdateRunningStatus(&summary, &startedAt)

	mockDeployService.On("RefreshActiveDeployment", mock.Anything, uint(1)).
		Return(refreshedDeployment, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint64(100), output.DeploymentID)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "running", output.Status)
	assert.Equal(t, "test-event-123", output.TektonEventID)
	assert.Equal(t, "test-run-123", output.TektonPipelineRunName)
	assert.Equal(t, "Deployment running", output.Summary)
	assert.NotEmpty(t, output.StartedAt)

	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_NoActiveDeployment(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	useCase := NewRefreshDeploymentUseCase(mockDeployService)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// No active deployment
	mockDeployService.On("RefreshActiveDeployment", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrDeploymentNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrDeploymentNotFound))

	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_ProjectNotFound(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	useCase := NewRefreshDeploymentUseCase(mockDeployService)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Project not found
	mockDeployService.On("RefreshActiveDeployment", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrProjectNotFound))

	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_RefreshError(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	useCase := NewRefreshDeploymentUseCase(mockDeployService)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	mockDeployService.On("RefreshActiveDeployment", mock.Anything, uint(1)).
		Return(nil, fmt.Errorf("kubernetes connection failed"))

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "kubernetes connection failed")

	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_WithAllOptionalFields(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	useCase := NewRefreshDeploymentUseCase(mockDeployService)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Create completed deployment with all optional fields
	refreshedDeployment := deployment.NewDeployment(1)
	refreshedDeployment.SetDeploymentID(100)
	eventID := "test-event-123"
	runName := "test-run-123"
	_ = refreshedDeployment.InitTektonInfo(&eventID, &runName)
	summary := "Deployment succeeded"
	startedAt := time.Now().Add(-5 * time.Minute)
	_ = refreshedDeployment.UpdateRunningStatus(&summary, &startedAt)
	finishedAt := time.Now()
	_ = refreshedDeployment.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, &summary, finishedAt)

	mockDeployService.On("RefreshActiveDeployment", mock.Anything, uint(1)).
		Return(refreshedDeployment, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint64(100), output.DeploymentID)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "success", output.Status)
	assert.Equal(t, "test-event-123", output.TektonEventID)
	assert.Equal(t, "test-run-123", output.TektonPipelineRunName)
	assert.Equal(t, "Deployment succeeded", output.Summary)
	assert.NotEmpty(t, output.StartedAt)
	assert.NotEmpty(t, output.FinishedAt)

	mockDeployService.AssertExpectations(t)
}
