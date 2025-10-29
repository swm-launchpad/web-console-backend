package deploy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/build"
)

// TestBuildAndDeployProject_ContainerConfigNotFound tests validation failure when no containers are found
func TestBuildAndDeployProject_ContainerConfigNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project exists (authorization check passes)
	project := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

	// Mock: GetContainerBuildConfig returns error
	mockContainerClient.On("GetContainerBuildConfig", ctx, projectID).
		Return(nil, projecterrors.ErrContainerConfigNotFound)

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrContainerConfigNotFound)
	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
}

// TestBuildAndDeployProject_EmptyContainers tests validation failure when container list is empty
func TestBuildAndDeployProject_EmptyContainers(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project exists (authorization check passes)
	project := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

	// Mock: GetContainerBuildConfig returns empty container list
	emptyConfig := &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{},
	}
	mockContainerClient.On("GetContainerBuildConfig", ctx, projectID).
		Return(emptyConfig, nil)

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrContainerConfigNotFound)
	mockProjectRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
}

// TestBuildAndDeployProject_ProjectAlreadyDeploying tests status validation
func TestBuildAndDeployProject_ProjectAlreadyDeploying(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project exists (authorization check passes)
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusDeploying)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)

	// Mock: GetContainerBuildConfig returns valid containers
	containerConfig := &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{
			{
				ProjectID:   projectID,
				ContainerID: 1,
				Name:        "test-container",
				Slug:        "test-slug",
			},
		},
	}
	mockContainerClient.On("GetContainerBuildConfig", ctx, projectID).
		Return(containerConfig, nil)

	// Mock: Project is already in 'deploying' state
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil).Once()

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrProjectAlreadyDeploying)
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployProject_Success tests the happy path (validation only, background goroutine not tested)
func TestBuildAndDeployProject_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	mockBuildPostProcessor := &build.MockBuildPostProcessor{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:          mockTxManager,
		projectRepo:        mockProjectRepo,
		containerClient:    mockContainerClient,
		buildOrchestrator:  mockBuildOrchestrator,
		buildPostProcessor: mockBuildPostProcessor,
		logger:             testLogger,
	}

	// Mock: Project exists (authorization check passes)
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)

	// Mock: GetContainerBuildConfig returns valid containers
	containerConfig := &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{
			{
				ProjectID:   projectID,
				ContainerID: 1,
				Name:        "test-container",
				Slug:        "test-slug",
			},
		},
	}
	mockContainerClient.On("GetContainerBuildConfig", ctx, projectID).
		Return(containerConfig, nil)

	// Mock: Project is in 'nothing' state and can transition to 'building'
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.MatchedBy(func(p *projectmodel.Project) bool {
		// Verify project status changed to 'building' or back to 'nothing'
		return p.OperationStatus() == value.ProjectOperationStatusBuilding ||
			p.OperationStatus() == value.ProjectOperationStatusNothing
	})).Return(nil)

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	mockContainerClient.AssertExpectations(t)
	// Note: We don't assert on projectRepo calls because the background goroutine
	// will make additional calls that we're not testing here

	// Note: Background goroutine execution is not tested here
	// Integration tests should cover the full build+deploy flow
	// Sleep briefly to allow goroutine to start (not a guarantee, just best effort for this unit test)
	time.Sleep(10 * time.Millisecond)
}

// TestBuildAndDeployProject_SaveFailure tests save failure during status update
func TestBuildAndDeployProject_SaveFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project exists (authorization check passes)
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)

	// Mock: GetContainerBuildConfig returns valid containers
	containerConfig := &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{
			{
				ProjectID:   projectID,
				ContainerID: 1,
				Name:        "test-container",
				Slug:        "test-slug",
			},
		},
	}
	mockContainerClient.On("GetContainerBuildConfig", ctx, projectID).
		Return(containerConfig, nil)

	// Mock: Save fails
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil).Once()

	saveError := errors.New("save failed")
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(saveError).Once()

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, saveError, err)
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployProject_ProjectNotFound tests project not found during transaction
func TestBuildAndDeployProject_ProjectNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(999)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project not found (authorization check fails)
	mockProjectRepo.On("FindByID", ctx, projectID).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Act
	err := service.BuildAndDeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, projecterrors.ErrProjectNotFound)
	mockProjectRepo.AssertExpectations(t)
	// Note: containerClient should NOT be called when project doesn't exist (authorization fails first)
}

// ==================== Background Flow Tests ====================

// TestBuildAndDeployInBackground_Success tests the successful background flow
// Note: This test only covers the build phase. The deployment phase is tested separately
// because deployProjectInternal performs long-running synchronous monitoring.
func TestBuildAndDeployInBackground_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	mockBuildPostProcessor := &build.MockBuildPostProcessor{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:          mockTxManager,
		projectRepo:        mockProjectRepo,
		containerClient:    mockContainerClient,
		buildOrchestrator:  mockBuildOrchestrator,
		buildPostProcessor: mockBuildPostProcessor,
		logger:             testLogger,
	}

	unifiedConfig := &dto.UnifiedContainerConfig{
		ProjectID: projectID,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID: 1,
				Name:        "test-container",
				Slug:        "test-slug",
				ImageName:   "test-image",
				ImageTag:    "latest",
				Port:        8080,
			},
		},
	}

	// Mock: GetUnifiedContainerConfig returns unified config
	// This allows us to test the build phase without having to mock the entire deployment flow
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(unifiedConfig, nil)

	// Mock: BuildOrchestrator returns success
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		return []*build.BuildResult{
			{
				BuildHistoryID:   1,
				Status:           "success",
				LatestCommitHash: "abc123",
				ImageTag:         "latest",
			},
		}, nil
	}

	// Mock: BuildPostProcessor succeeds
	mockBuildPostProcessor.UpdateContainerAfterBuildFunc = func(ctx context.Context, containerID uint, result *build.BuildResult, snapshot *dto.BuildContainerInfo) error {
		return nil
	}

	// Mock: handleBuildError will be called (since GetContainerConfig fails)
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
	// Note: The deployment phase is not tested here due to its long-running synchronous nature.
	// Integration tests should cover the full build+deploy flow.
}

// TestBuildAndDeployInBackground_BuildOrchestrationFailure tests build orchestration failure
func TestBuildAndDeployInBackground_BuildOrchestrationFailure(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:         mockTxManager,
		projectRepo:       mockProjectRepo,
		containerClient:   mockContainerClient,
		buildOrchestrator: mockBuildOrchestrator,
		logger:            testLogger,
	}

	unifiedConfig := &dto.UnifiedContainerConfig{
		ProjectID: projectID,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID: 1,
				ImageName:   "test-image",
				ImageTag:    "latest",
				Port:        8080,
			},
		},
	}
	// Mock: GetUnifiedContainerConfig returns unified config
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(unifiedConfig, nil)

	// Mock: BuildOrchestrator returns error
	orchError := errors.New("orchestration failed")
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		return nil, orchError
	}

	// Mock: handleBuildError will try to reset project status
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployInBackground_BuildFailures tests when builds fail
func TestBuildAndDeployInBackground_BuildFailures(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:         mockTxManager,
		projectRepo:       mockProjectRepo,
		containerClient:   mockContainerClient,
		buildOrchestrator: mockBuildOrchestrator,
		logger:            testLogger,
	}

	unifiedConfig := &dto.UnifiedContainerConfig{
		ProjectID: projectID,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID: 1,
				ImageName:   "test-image-1",
				ImageTag:    "latest",
				Port:        8080,
			},
			{
				ContainerID: 2,
				ImageName:   "test-image-2",
				ImageTag:    "latest",
				Port:        8081,
			},
		},
	}
	// Mock: GetUnifiedContainerConfig returns unified config
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(unifiedConfig, nil)

	// Mock: BuildOrchestrator returns with one failure
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		return []*build.BuildResult{
			{BuildHistoryID: 1, Status: "success"},
			{BuildHistoryID: 2, Status: "failed"}, // One failed
		}, nil
	}

	// Mock: handleBuildError will try to reset project status
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployInBackground_ContainerChangedDuringBuild tests the snapshot drift scenario
func TestBuildAndDeployInBackground_ContainerChangedDuringBuild(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	mockBuildPostProcessor := &build.MockBuildPostProcessor{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:          mockTxManager,
		projectRepo:        mockProjectRepo,
		containerClient:    mockContainerClient,
		buildOrchestrator:  mockBuildOrchestrator,
		buildPostProcessor: mockBuildPostProcessor,
		logger:             testLogger,
	}

	unifiedConfig := &dto.UnifiedContainerConfig{
		ProjectID: projectID,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID: 1,
				ImageName:   "test-image",
				ImageTag:    "latest",
				Port:        8080,
			},
		},
	}
	// Mock: GetUnifiedContainerConfig returns unified config
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(unifiedConfig, nil)

	// Mock: BuildOrchestrator succeeds
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		return []*build.BuildResult{
			{BuildHistoryID: 1, Status: "success", LatestCommitHash: "abc123"},
		}, nil
	}

	// Mock: BuildPostProcessor returns ErrContainerChangedDuringBuild
	mockBuildPostProcessor.UpdateContainerAfterBuildFunc = func(ctx context.Context, containerID uint, result *build.BuildResult, snapshot *dto.BuildContainerInfo) error {
		return projecterrors.ErrContainerChangedDuringBuild
	}

	// Mock: handleBuildError will try to reset project status
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployInBackground_PostProcessorError tests other post-processor errors
func TestBuildAndDeployInBackground_PostProcessorError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	mockBuildPostProcessor := &build.MockBuildPostProcessor{}
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:          mockTxManager,
		projectRepo:        mockProjectRepo,
		containerClient:    mockContainerClient,
		buildOrchestrator:  mockBuildOrchestrator,
		buildPostProcessor: mockBuildPostProcessor,
		logger:             testLogger,
	}

	unifiedConfig := &dto.UnifiedContainerConfig{
		ProjectID: projectID,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID: 1,
				ImageName:   "test-image",
				ImageTag:    "latest",
				Port:        8080,
			},
		},
	}
	// Mock: GetUnifiedContainerConfig returns unified config
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(unifiedConfig, nil)

	// Mock: BuildOrchestrator succeeds
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		return []*build.BuildResult{
			{BuildHistoryID: 1, Status: "success"},
		}, nil
	}

	// Mock: BuildPostProcessor returns generic error
	mockBuildPostProcessor.UpdateContainerAfterBuildFunc = func(ctx context.Context, containerID uint, result *build.BuildResult, snapshot *dto.BuildContainerInfo) error {
		return errors.New("database error")
	}

	// Mock: handleBuildError will try to reset project status
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

// TestBuildAndDeployInBackground_ContainersDeletedAfterStatusFlip tests the race condition where
// containers are deleted between the initial validation and the background goroutine execution
func TestBuildAndDeployInBackground_ContainersDeletedAfterStatusFlip(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()
	projectID := uint(1)

	// Setup mocks
	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockBuildOrchestrator := &build.MockBuildOrchestrator{}
	mockBuildPostProcessor := &build.MockBuildPostProcessor{}

	service := &deployService{
		txManager:          mockTxManager,
		projectRepo:        mockProjectRepo,
		containerClient:    mockContainerClient,
		buildOrchestrator:  mockBuildOrchestrator,
		buildPostProcessor: mockBuildPostProcessor,
		logger:             testLogger,
	}

	// Mock: GetUnifiedContainerConfig returns empty list (containers deleted after status flip)
	mockContainerClient.On("GetUnifiedContainerConfig", mock.Anything, projectID).
		Return(&dto.UnifiedContainerConfig{
			ProjectID:  projectID,
			Containers: []dto.UnifiedContainerInfo{}, // Empty list
		}, nil)

	// Note: We fail early on empty containers

	// Mock: BuildOrchestrator should NOT be called - fail the test if it is
	mockBuildOrchestrator.BuildAndWaitFunc = func(ctx context.Context, pid uint, containers []*dto.BuildContainerInfo) ([]*build.BuildResult, error) {
		t.Fatalf("BuildAndWait should not be called when containers are deleted after status flip")
		return nil, nil
	}

	// Mock: handleBuildError will try to reset project status
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusBuilding)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).
		Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).
		Return(nil)

	// Act
	service.buildAndDeployInBackground(ctx, projectID)

	// Assert: handleBuildError should be called, build should not proceed
	mockContainerClient.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
	// Note: If the guard regresses, BuildAndWait will be called and t.Fatalf will trigger
}
