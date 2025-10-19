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
	domainrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestRefreshDeploymentUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Create mock active deployment
	activeDeployment := deployment.NewDeployment(1)
	activeDeployment.SetDeploymentID(100)
	eventID := "test-event-123"
	_ = activeDeployment.InitTektonInfo(&eventID, nil)

	// Create refreshed deployment with updated status
	refreshedDeployment := deployment.NewDeployment(1)
	refreshedDeployment.SetDeploymentID(100)
	_ = refreshedDeployment.InitTektonInfo(&eventID, nil)
	runName := "test-run-123"
	_ = refreshedDeployment.InitTektonInfo(nil, &runName)
	summary := "Deployment running"
	startedAt := time.Now()
	_ = refreshedDeployment.UpdateRunningStatus(&summary, &startedAt)

	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return([]*deployment.Deployment{activeDeployment}, nil)
	mockDeployService.On("RefreshDeploymentStatus", mock.Anything, uint64(100)).
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

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_NoActiveDeployment(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// No active deployments
	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return([]*deployment.Deployment{}, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrDeploymentNotFound))

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_MultipleActiveDeployments(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Multiple active deployments (invariant violation)
	activeDeployment1 := deployment.NewDeployment(1)
	activeDeployment1.SetDeploymentID(100)
	activeDeployment2 := deployment.NewDeployment(1)
	activeDeployment2.SetDeploymentID(101)

	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return([]*deployment.Deployment{activeDeployment1, activeDeployment2}, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "invariant violation")
	assert.Contains(t, err.Error(), "2 active deployments")

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_RepositoryError(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Repository returns error
	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrDatabaseOperation)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrDatabaseOperation))

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_RefreshError(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Create mock active deployment
	activeDeployment := deployment.NewDeployment(1)
	activeDeployment.SetDeploymentID(100)

	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return([]*deployment.Deployment{activeDeployment}, nil)
	mockDeployService.On("RefreshDeploymentStatus", mock.Anything, uint64(100)).
		Return(nil, fmt.Errorf("kubernetes connection failed"))

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "kubernetes connection failed")

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}

func TestRefreshDeploymentUseCase_Execute_WithAllOptionalFields(t *testing.T) {
	// Arrange
	mockDeployService := new(service.MockDeployService)
	mockDeploymentRepo := new(domainrepo.MockDeploymentRepository)
	useCase := NewRefreshDeploymentUseCase(mockDeployService, mockDeploymentRepo)

	input := RefreshDeploymentInput{
		ProjectID: 1,
	}

	// Create mock active deployment
	activeDeployment := deployment.NewDeployment(1)
	activeDeployment.SetDeploymentID(100)

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

	mockDeploymentRepo.On("FindActiveDeploymentsByProjectID", mock.Anything, uint(1)).
		Return([]*deployment.Deployment{activeDeployment}, nil)
	mockDeployService.On("RefreshDeploymentStatus", mock.Anything, uint64(100)).
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

	mockDeploymentRepo.AssertExpectations(t)
	mockDeployService.AssertExpectations(t)
}
