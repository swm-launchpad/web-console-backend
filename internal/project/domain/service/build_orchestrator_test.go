package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

// Test mock implementations

type mockOrchestratorBuildHistoryRepo struct {
	createFunc func(ctx context.Context, b *build_history.BuildHistory) error
	saveFunc   func(ctx context.Context, b *build_history.BuildHistory) error
}

func (m *mockOrchestratorBuildHistoryRepo) Create(ctx context.Context, b *build_history.BuildHistory) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, b)
	}
	// Default: set ID and return success
	b.SetBuildHistoryID(1)
	return nil
}

func (m *mockOrchestratorBuildHistoryRepo) Save(ctx context.Context, b *build_history.BuildHistory) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, b)
	}
	return nil
}

func (m *mockOrchestratorBuildHistoryRepo) FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockOrchestratorBuildHistoryRepo) FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockOrchestratorBuildHistoryRepo) FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockOrchestratorBuildHistoryRepo) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockOrchestratorBuildHistoryRepo) FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

type mockOrchestratorBuildService struct {
	buildContainerFunc func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error)
}

func (m *mockOrchestratorBuildService) BuildContainer(
	ctx context.Context,
	buildHistoryID uint,
	container *dto.BuildContainerInfo,
) (*BuildResult, error) {
	if m.buildContainerFunc != nil {
		return m.buildContainerFunc(ctx, buildHistoryID, container)
	}
	// Default: return success
	return &BuildResult{
		BuildHistoryID:   buildHistoryID,
		Status:           "success",
		LatestCommitHash: "abc123def456",
		ImageTag:         "latest",
		ShouldBuild:      true,
	}, nil
}

// Test cases

func TestBuildOrchestrator_BuildAndWait_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryIDCounter := uint(0)
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			buildHistoryIDCounter++
			b.SetBuildHistoryID(buildHistoryIDCounter)
			return nil
		},
	}

	mockBuildService := &mockOrchestratorBuildService{
		buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
			return &BuildResult{
				BuildHistoryID:   buildHistoryID,
				Status:           "success",
				LatestCommitHash: "commit-" + container.Name,
				ImageTag:         "latest",
				ShouldBuild:      true,
			}, nil
		},
	}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data
	containers := []*dto.BuildContainerInfo{
		{
			ProjectID:        1,
			ContainerID:      10,
			Name:             "web",
			Slug:             "web",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
		{
			ProjectID:        1,
			ContainerID:      11,
			Name:             "api",
			Slug:             "api",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check first result
	assert.Equal(t, uint(1), results[0].BuildHistoryID)
	assert.Equal(t, "success", results[0].Status)
	assert.Equal(t, "commit-web", results[0].LatestCommitHash)
	assert.True(t, results[0].ShouldBuild)

	// Check second result
	assert.Equal(t, uint(2), results[1].BuildHistoryID)
	assert.Equal(t, "success", results[1].Status)
	assert.Equal(t, "commit-api", results[1].LatestCommitHash)
	assert.True(t, results[1].ShouldBuild)
}

func TestBuildOrchestrator_BuildAndWait_PartialFailure(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryIDCounter := uint(0)
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			buildHistoryIDCounter++
			b.SetBuildHistoryID(buildHistoryIDCounter)
			return nil
		},
	}

	mockBuildService := &mockOrchestratorBuildService{
		buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
			// First container succeeds, second fails
			if container.Name == "web" {
				return &BuildResult{
					BuildHistoryID:   buildHistoryID,
					Status:           "success",
					LatestCommitHash: "abc123",
					ImageTag:         "latest",
					ShouldBuild:      true,
				}, nil
			}
			return &BuildResult{
				BuildHistoryID: buildHistoryID,
				Status:         "failed",
				ErrorMessage:   "build failed",
				ShouldBuild:    true,
			}, nil
		},
	}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data
	containers := []*dto.BuildContainerInfo{
		{
			ProjectID:        1,
			ContainerID:      10,
			Name:             "web",
			Slug:             "web",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
		{
			ProjectID:        1,
			ContainerID:      11,
			Name:             "api",
			Slug:             "api",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check first result (success)
	assert.Equal(t, "success", results[0].Status)
	assert.Equal(t, "abc123", results[0].LatestCommitHash)

	// Check second result (failed)
	assert.Equal(t, "failed", results[1].Status)
	assert.Equal(t, "build failed", results[1].ErrorMessage)
}

func TestBuildOrchestrator_BuildAndWait_BuildServiceError(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryIDCounter := uint(0)
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			buildHistoryIDCounter++
			b.SetBuildHistoryID(buildHistoryIDCounter)
			return nil
		},
	}

	buildErr := errors.New("build service error")
	mockBuildService := &mockOrchestratorBuildService{
		buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
			// Return error for one container
			if container.Name == "web" {
				return nil, buildErr
			}
			return &BuildResult{
				BuildHistoryID:   buildHistoryID,
				Status:           "success",
				LatestCommitHash: "abc123",
				ImageTag:         "latest",
				ShouldBuild:      true,
			}, nil
		},
	}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data
	containers := []*dto.BuildContainerInfo{
		{
			ProjectID:        1,
			ContainerID:      10,
			Name:             "web",
			Slug:             "web",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
		{
			ProjectID:        1,
			ContainerID:      11,
			Name:             "api",
			Slug:             "api",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify - BuildAndWait should not return error, but results should contain error info
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check first result (error converted to failed BuildResult)
	assert.Equal(t, "failed", results[0].Status)
	assert.Equal(t, buildErr.Error(), results[0].ErrorMessage)

	// Check second result (success)
	assert.Equal(t, "success", results[1].Status)
}

func TestBuildOrchestrator_BuildAndWait_EmptyContainers(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	mockRepo := &mockOrchestratorBuildHistoryRepo{}
	mockBuildService := &mockOrchestratorBuildService{}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Execute with empty containers
	results, err := orchestrator.BuildAndWait(ctx, 1, []*dto.BuildContainerInfo{})

	// Verify
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestBuildOrchestrator_BuildAndWait_BuildHistoryCreationFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	repoErr := errors.New("database error")
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return repoErr
		},
	}

	mockBuildService := &mockOrchestratorBuildService{}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data
	containers := []*dto.BuildContainerInfo{
		{
			ProjectID:        1,
			ContainerID:      10,
			Name:             "web",
			Slug:             "web",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		},
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify
	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to create build histories")
}

func TestBuildOrchestrator_BuildAndWait_MultipleContainers(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryIDCounter := uint(0)
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			buildHistoryIDCounter++
			b.SetBuildHistoryID(buildHistoryIDCounter)
			return nil
		},
	}

	mockBuildService := &mockOrchestratorBuildService{}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data - 5 containers
	containers := make([]*dto.BuildContainerInfo, 5)
	for i := 0; i < 5; i++ {
		containers[i] = &dto.BuildContainerInfo{
			ProjectID:        1,
			ContainerID:      uint(10 + i),
			Name:             "container-" + string(rune('a'+i)),
			Slug:             "container-" + string(rune('a'+i)),
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		}
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify
	require.NoError(t, err)
	require.Len(t, results, 5)

	// Verify all results
	for i := 0; i < 5; i++ {
		assert.Equal(t, uint(i+1), results[i].BuildHistoryID)
		assert.Equal(t, "success", results[i].Status)
	}
}

func TestBuildOrchestrator_BuildAndWait_OrderPreserved(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryIDCounter := uint(0)
	mockRepo := &mockOrchestratorBuildHistoryRepo{
		createFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			buildHistoryIDCounter++
			b.SetBuildHistoryID(buildHistoryIDCounter)
			return nil
		},
	}

	mockBuildService := &mockOrchestratorBuildService{
		buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
			// Return container name in commit hash to verify ordering
			return &BuildResult{
				BuildHistoryID:   buildHistoryID,
				Status:           "success",
				LatestCommitHash: "commit-" + container.Name,
				ImageTag:         "latest",
				ShouldBuild:      true,
			}, nil
		},
	}

	orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

	// Test data
	containers := []*dto.BuildContainerInfo{
		{ProjectID: 1, ContainerID: 10, Name: "first", Slug: "first", GitRepositoryURL: "url", GitBranch: "main", NeedsBuild: true},
		{ProjectID: 1, ContainerID: 11, Name: "second", Slug: "second", GitRepositoryURL: "url", GitBranch: "main", NeedsBuild: true},
		{ProjectID: 1, ContainerID: 12, Name: "third", Slug: "third", GitRepositoryURL: "url", GitBranch: "main", NeedsBuild: true},
	}

	// Execute
	results, err := orchestrator.BuildAndWait(ctx, 1, containers)

	// Verify order is preserved
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "commit-first", results[0].LatestCommitHash)
	assert.Equal(t, "commit-second", results[1].LatestCommitHash)
	assert.Equal(t, "commit-third", results[2].LatestCommitHash)
}

func TestBuildOrchestrator_ErrorHandling(t *testing.T) {
	// Test that BuildOrchestrator preserves BuildResult even when error is present
	// and distinguishes context cancellation from real failures
	testLogger := logger.NewForTest()

	t.Run("Preserve BuildResult when returned with error", func(t *testing.T) {
		// BuildService often returns both BuildResult and error for terminal states
		// (e.g., monitoring timeout, terminal failure with metadata)
		// Orchestrator must preserve the BuildResult, not throw it away

		mockRepo := &mockOrchestratorBuildHistoryRepo{
			createFunc: func(ctx context.Context, buildHistory *build_history.BuildHistory) error {
				buildHistory.SetBuildHistoryID(1)
				return nil
			},
		}

		expectedResult := &BuildResult{
			BuildHistoryID:   1,
			Status:           "failed",
			ErrorMessage:     "Tekton reported failure",
			LatestCommitHash: "abc123", // Important metadata
			ShouldBuild:      true,
		}

		mockBuildService := &mockOrchestratorBuildService{
			buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
				// Return both result and error (common pattern in BuildService)
				return expectedResult, fmt.Errorf("build failed")
			},
		}

		orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

		containers := []*dto.BuildContainerInfo{
			{ContainerID: 1, Name: "test", Slug: "test"},
		}

		results, err := orchestrator.BuildAndWait(context.Background(), 10, containers)

		// Should not return error (per current design - all errors absorbed)
		assert.NoError(t, err)
		require.Len(t, results, 1)

		// CRITICAL: Must preserve the original BuildResult, not create generic fallback
		assert.Equal(t, expectedResult.Status, results[0].Status)
		assert.Equal(t, expectedResult.ErrorMessage, results[0].ErrorMessage)
		assert.Equal(t, expectedResult.LatestCommitHash, results[0].LatestCommitHash)
		assert.Equal(t, expectedResult.ShouldBuild, results[0].ShouldBuild)
	})

	t.Run("Create fallback for nil BuildResult with regular error", func(t *testing.T) {
		// Only when service returns nil should we create a fallback

		mockRepo := &mockOrchestratorBuildHistoryRepo{
			createFunc: func(ctx context.Context, buildHistory *build_history.BuildHistory) error {
				buildHistory.SetBuildHistoryID(1)
				return nil
			},
		}

		mockBuildService := &mockOrchestratorBuildService{
			buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
				// Return nil result with error
				return nil, fmt.Errorf("unexpected infrastructure error")
			},
		}

		orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

		containers := []*dto.BuildContainerInfo{
			{ContainerID: 1, Name: "test", Slug: "test"},
		}

		results, err := orchestrator.BuildAndWait(context.Background(), 10, containers)

		assert.NoError(t, err)
		require.Len(t, results, 1)

		// Should create fallback with generic "failed" status
		assert.Equal(t, "failed", results[0].Status)
		assert.Contains(t, results[0].ErrorMessage, "unexpected infrastructure error")
		assert.True(t, results[0].ShouldBuild)
	})

	t.Run("Handle context.Canceled as non-terminal", func(t *testing.T) {
		// PR #7 Option B: context cancellation returns nil result
		// Orchestrator must not convert this to "failed" status
		// BuildHistory should remain in non-terminal state

		mockRepo := &mockOrchestratorBuildHistoryRepo{
			createFunc: func(ctx context.Context, buildHistory *build_history.BuildHistory) error {
				buildHistory.SetBuildHistoryID(1)
				return nil
			},
		}

		mockBuildService := &mockOrchestratorBuildService{
			buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
				// Simulate PR #7 Option B behavior
				return nil, context.Canceled
			},
		}

		orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

		containers := []*dto.BuildContainerInfo{
			{ContainerID: 1, Name: "test", Slug: "test"},
		}

		results, err := orchestrator.BuildAndWait(context.Background(), 10, containers)

		assert.NoError(t, err)
		require.Len(t, results, 1)

		// CRITICAL: Must return nil to indicate non-terminal state
		// Not "failed" status which would undo PR #7 design
		assert.Nil(t, results[0], "context cancellation should result in nil BuildResult, not 'failed' status")
	})

	t.Run("Handle context.DeadlineExceeded as non-terminal", func(t *testing.T) {
		mockRepo := &mockOrchestratorBuildHistoryRepo{
			createFunc: func(ctx context.Context, buildHistory *build_history.BuildHistory) error {
				buildHistory.SetBuildHistoryID(1)
				return nil
			},
		}

		mockBuildService := &mockOrchestratorBuildService{
			buildContainerFunc: func(ctx context.Context, buildHistoryID uint, container *dto.BuildContainerInfo) (*BuildResult, error) {
				// Simulate PR #7 Option B behavior
				return nil, context.DeadlineExceeded
			},
		}

		orchestrator := NewBuildOrchestrator(mockRepo, mockBuildService, testLogger)

		containers := []*dto.BuildContainerInfo{
			{ContainerID: 1, Name: "test", Slug: "test"},
		}

		results, err := orchestrator.BuildAndWait(context.Background(), 10, containers)

		assert.NoError(t, err)
		require.Len(t, results, 1)

		// Same as Canceled - must preserve non-terminal state
		assert.Nil(t, results[0], "deadline exceeded should result in nil BuildResult, not 'failed' status")
	})
}
