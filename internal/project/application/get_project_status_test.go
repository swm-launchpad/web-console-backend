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
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// createTestProject creates a test project with the given status
func createTestProject(projectID uint, operationStatus value.ProjectOperationStatus) *projectmodel.Project {
	slug, _ := value.NewProjectSlug("p2025011812000012345678")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	plan, _ := value.NewPlan("eco")
	now := time.Now()

	return projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		value.ProjectStatusActive,
		operationStatus,
		nil, // activeDeploymentID
		&plan,
		*limits,
		now,
		now,
		false,
		nil,
	)
}

// createTestProjectWithDeployment creates a test project with an active deployment
func createTestProjectWithDeployment(projectID uint, operationStatus value.ProjectOperationStatus, deploymentID uint) *projectmodel.Project {
	slug, _ := value.NewProjectSlug("p2025011812000012345678")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	plan, _ := value.NewPlan("eco")
	now := time.Now()

	return projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		value.ProjectStatusActive,
		operationStatus,
		&deploymentID,
		&plan,
		*limits,
		now,
		now,
		false,
		nil,
	)
}

func TestGetProjectStatusUseCase_Execute_NothingStatus(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "nothing" status
	proj := createTestProject(1, value.ProjectOperationStatusNothing)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock container client response
	containers := []dto.ContainerBasicInfo{
		{ContainerID: 1, Name: "backend"},
	}
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).Return(containers, nil)

	// Mock build history for the container
	bh1 := build_history.NewBuildHistory(1)
	bh1.SetBuildHistoryID(10)
	summary1 := "Build completed"
	startedAt1 := time.Now()
	_ = bh1.UpdateRunningStatus(&summary1, &startedAt1)
	finishedAt1 := time.Now().Add(time.Minute)
	_ = bh1.UpdateCompleteStatus(build_history.BuildHistoryStatusSuccess, &summary1, nil, finishedAt1)
	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(1)).Return(bh1, nil)

	// Mock deployment (return not found - no deployment yet)
	mockDeploymentRepo.On("FindLatestByProjectID", mock.Anything, uint(1)).Return(nil, projecterrors.ErrDeploymentNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "nothing", output.OperationStatus)
	assert.NotNil(t, output.BuildStatuses)
	assert.Len(t, output.BuildStatuses, 1)
	assert.Equal(t, uint(1), output.BuildStatuses[0].ContainerID)
	assert.Equal(t, "backend", output.BuildStatuses[0].ContainerName)
	assert.Equal(t, "success", output.BuildStatuses[0].Status)
	assert.Nil(t, output.DeploymentStatus)

	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_BuildingStatus(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "building" status
	proj := createTestProject(1, value.ProjectOperationStatusBuilding)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock container client response
	containers := []dto.ContainerBasicInfo{
		{ContainerID: 1, Name: "backend"},
		{ContainerID: 2, Name: "mysql"},
	}
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).Return(containers, nil)

	// Mock build history for each container
	bh1 := build_history.NewBuildHistory(1)
	bh1.SetBuildHistoryID(10)
	eventID1 := "build-event-1"
	runName1 := "build-run-1"
	_ = bh1.InitTektonInfo(&eventID1, &runName1)
	summary1 := "Building..."
	startedAt1 := time.Now()
	_ = bh1.UpdateRunningStatus(&summary1, &startedAt1)

	bh2 := build_history.NewBuildHistory(2)
	bh2.SetBuildHistoryID(11)
	eventID2 := "build-event-2"
	runName2 := "build-run-2"
	commitHash2 := "abc123def456"
	_ = bh2.InitTektonInfo(&eventID2, &runName2)
	finishedAt2 := time.Now()
	_ = bh2.UpdateCompleteStatus(build_history.BuildHistoryStatusSuccess, nil, &commitHash2, finishedAt2)

	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(1)).Return(bh1, nil)
	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(2)).Return(bh2, nil)

	// Mock deployment (previous completed deployment exists)
	d := deployment.NewDeployment(1, nil, nil)
	d.SetDeploymentID(50)
	deployEventID := "deploy-event-50"
	deployRunName := "deploy-run-50"
	_ = d.InitTektonInfo(&deployEventID, &deployRunName)
	deploySummary := "Deployment completed"
	deployStartedAt := time.Now().Add(-2 * time.Hour)
	_ = d.UpdateRunningStatus(&deploySummary, &deployStartedAt)
	deployFinishedAt := time.Now().Add(-90 * time.Minute)
	_ = d.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, &deploySummary, deployFinishedAt)
	mockDeploymentRepo.On("FindLatestByProjectID", mock.Anything, uint(1)).Return(d, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "building", output.OperationStatus)
	assert.Len(t, output.BuildStatuses, 2)

	// Check first build status (running)
	assert.Equal(t, uint64(10), output.BuildStatuses[0].BuildHistoryID)
	assert.Equal(t, uint(1), output.BuildStatuses[0].ContainerID)
	assert.Equal(t, "backend", output.BuildStatuses[0].ContainerName)
	assert.Equal(t, "running", output.BuildStatuses[0].Status)
	assert.NotEmpty(t, output.BuildStatuses[0].StartedAt)
	assert.Empty(t, output.BuildStatuses[0].FinishedAt)

	// Check second build status (success)
	assert.Equal(t, uint64(11), output.BuildStatuses[1].BuildHistoryID)
	assert.Equal(t, uint(2), output.BuildStatuses[1].ContainerID)
	assert.Equal(t, "mysql", output.BuildStatuses[1].ContainerName)
	assert.Equal(t, "success", output.BuildStatuses[1].Status)
	assert.Equal(t, "abc123def456", output.BuildStatuses[1].GitCommitHash)
	assert.NotEmpty(t, output.BuildStatuses[1].FinishedAt)

	// Deployment status should now be returned in building case
	assert.NotNil(t, output.DeploymentStatus)
	assert.Equal(t, uint64(50), output.DeploymentStatus.DeploymentID)
	assert.Equal(t, uint(1), output.DeploymentStatus.ProjectID)
	assert.Equal(t, "success", output.DeploymentStatus.Status)
	assert.Equal(t, "Deployment completed", output.DeploymentStatus.Summary)
	assert.NotEmpty(t, output.DeploymentStatus.StartedAt)
	assert.NotEmpty(t, output.DeploymentStatus.FinishedAt)

	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_DeployingStatus(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "deploying" status and active deployment
	proj := createTestProjectWithDeployment(1, value.ProjectOperationStatusDeploying, 100)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock deployment
	d := deployment.NewDeployment(1, nil, nil)
	d.SetDeploymentID(100)
	eventID := "deploy-event-123"
	runName := "deploy-run-123"
	_ = d.InitTektonInfo(&eventID, &runName)
	summary := "Deploying..."
	startedAt := time.Now()
	_ = d.UpdateRunningStatus(&summary, &startedAt)

	mockDeploymentRepo.On("FindByID", mock.Anything, uint(100)).Return(d, nil)

	// Mock container client response for build statuses
	containers := []dto.ContainerBasicInfo{
		{ContainerID: 1, Name: "backend"},
	}
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).Return(containers, nil)

	// Mock build history for the container
	bh1 := build_history.NewBuildHistory(1)
	bh1.SetBuildHistoryID(10)
	buildSummary := "Build completed"
	buildStartedAt := time.Now().Add(-time.Hour)
	_ = bh1.UpdateRunningStatus(&buildSummary, &buildStartedAt)
	buildFinishedAt := time.Now().Add(-30 * time.Minute)
	_ = bh1.UpdateCompleteStatus(build_history.BuildHistoryStatusSuccess, &buildSummary, nil, buildFinishedAt)
	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(1)).Return(bh1, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "deploying", output.OperationStatus)

	// Build statuses should now be returned in deploying case
	assert.NotNil(t, output.BuildStatuses)
	assert.Len(t, output.BuildStatuses, 1)
	assert.Equal(t, uint(1), output.BuildStatuses[0].ContainerID)
	assert.Equal(t, "backend", output.BuildStatuses[0].ContainerName)
	assert.Equal(t, "success", output.BuildStatuses[0].Status)

	assert.NotNil(t, output.DeploymentStatus)
	assert.Equal(t, uint64(100), output.DeploymentStatus.DeploymentID)
	assert.Equal(t, uint(1), output.DeploymentStatus.ProjectID)
	assert.Equal(t, "running", output.DeploymentStatus.Status)
	assert.Equal(t, "Deploying...", output.DeploymentStatus.Summary)
	assert.NotEmpty(t, output.DeploymentStatus.StartedAt)
	assert.Empty(t, output.DeploymentStatus.FinishedAt)

	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_ProjectNotFound(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 999,
	}

	mockProjectRepo.On("FindByID", mock.Anything, uint(999)).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.True(t, errors.Is(err, projecterrors.ErrProjectNotFound))

	mockProjectRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_BuildingWithNoContainers(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "building" status
	proj := createTestProject(1, value.ProjectOperationStatusBuilding)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock container client error
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrContainerConfigNotFound)

	// Mock deployment (no previous deployment)
	mockDeploymentRepo.On("FindLatestByProjectID", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrDeploymentNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert - Should not fail, just return empty build statuses
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "building", output.OperationStatus)
	assert.Empty(t, output.BuildStatuses)
	assert.Nil(t, output.DeploymentStatus)

	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_BuildingWithNoBuildHistory(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "building" status
	proj := createTestProject(1, value.ProjectOperationStatusBuilding)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock container client response
	containers := []dto.ContainerBasicInfo{
		{ContainerID: 1, Name: "backend"},
	}
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).Return(containers, nil)

	// Mock build history not found (container never built)
	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrBuildHistoryNotFound)

	// Mock deployment (no previous deployment)
	mockDeploymentRepo.On("FindLatestByProjectID", mock.Anything, uint(1)).
		Return(nil, projecterrors.ErrDeploymentNotFound)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert - Should return 'running' status for containers without build history in 'building' state
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "building", output.OperationStatus)
	assert.NotNil(t, output.BuildStatuses)
	assert.Len(t, output.BuildStatuses, 1)
	assert.Equal(t, uint(1), output.BuildStatuses[0].ContainerID)
	assert.Equal(t, "backend", output.BuildStatuses[0].ContainerName)
	assert.Equal(t, "running", output.BuildStatuses[0].Status) // 'building' with no history shows 'running'
	assert.Nil(t, output.DeploymentStatus)

	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestGetProjectStatusUseCase_Execute_DeployingWithoutActiveDeployment(t *testing.T) {
	// Arrange
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	useCase := NewGetProjectStatusUseCase(
		mockProjectRepo,
		mockDeploymentRepo,
		mockBuildHistoryRepo,
		mockContainerClient,
		testLogger,
	)

	input := GetProjectStatusInput{
		ProjectID: 1,
	}

	// Create project with "deploying" status but NO active deployment
	proj := createTestProject(1, value.ProjectOperationStatusDeploying)

	mockProjectRepo.On("FindByID", mock.Anything, uint(1)).Return(proj, nil)

	// Mock container client response for build statuses
	containers := []dto.ContainerBasicInfo{
		{ContainerID: 1, Name: "backend"},
	}
	mockContainerClient.On("GetContainerIDsByProjectID", mock.Anything, uint(1)).Return(containers, nil)

	// Mock build history for the container
	bh1 := build_history.NewBuildHistory(1)
	bh1.SetBuildHistoryID(10)
	buildSummary := "Build completed"
	buildStartedAt := time.Now().Add(-time.Hour)
	_ = bh1.UpdateRunningStatus(&buildSummary, &buildStartedAt)
	buildFinishedAt := time.Now().Add(-30 * time.Minute)
	_ = bh1.UpdateCompleteStatus(build_history.BuildHistoryStatusSuccess, &buildSummary, nil, buildFinishedAt)
	mockBuildHistoryRepo.On("FindLatestByContainerID", mock.Anything, uint(1)).Return(bh1, nil)

	// Act
	output, err := useCase.Execute(context.Background(), input)

	// Assert - Should return build statuses even without active deployment
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, uint(1), output.ProjectID)
	assert.Equal(t, "deploying", output.OperationStatus)
	assert.Nil(t, output.DeploymentStatus) // No active deployment
	assert.NotNil(t, output.BuildStatuses) // But build statuses should be present
	assert.Len(t, output.BuildStatuses, 1)
	assert.Equal(t, "success", output.BuildStatuses[0].Status)

	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
}
