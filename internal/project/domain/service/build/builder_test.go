package build

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

// Mock implementations for testing

type mockBuildHistoryRepository struct {
	findByIDFunc func(ctx context.Context, id uint) (*build_history.BuildHistory, error)
	saveFunc     func(ctx context.Context, b *build_history.BuildHistory) error
}

func (m *mockBuildHistoryRepository) Create(ctx context.Context, b *build_history.BuildHistory) error {
	return nil
}

func (m *mockBuildHistoryRepository) Save(ctx context.Context, b *build_history.BuildHistory) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, b)
	}
	return nil
}

func (m *mockBuildHistoryRepository) FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, buildHistoryID)
	}
	return build_history.NewBuildHistory(1), nil
}

func (m *mockBuildHistoryRepository) FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockBuildHistoryRepository) FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockBuildHistoryRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockBuildHistoryRepository) FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

type mockTektonBuildClient struct {
	triggerBuildFunc func(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error)
}

func (m *mockTektonBuildClient) TriggerBuild(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
	if m.triggerBuildFunc != nil {
		return m.triggerBuildFunc(ctx, request)
	}
	return &dto.TektonBuildResponse{EventID: "test-event-123"}, nil
}

type mockKubeBuildClient struct {
	getPipelineRunStatusFunc         func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error)
	findPipelineRunNameByEventIDFunc func(ctx context.Context, eventID string) (string, error)
}

func (m *mockKubeBuildClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
	if m.getPipelineRunStatusFunc != nil {
		return m.getPipelineRunStatusFunc(ctx, pipelineRunName)
	}
	return &dto.PipelineRun{
		Status:  "True",
		Reason:  "Succeeded",
		Message: "Build completed successfully",
		Results: map[string]string{
			"latest_commit_hash": "abc123",
			"image_tag":          "latest",
			"should_build":       "true",
		},
	}, nil
}

func (m *mockKubeBuildClient) FindPipelineRunNameByEventID(ctx context.Context, eventID string) (string, error) {
	if m.findPipelineRunNameByEventIDFunc != nil {
		return m.findPipelineRunNameByEventIDFunc(ctx, eventID)
	}
	return "test-pipeline-run-123", nil
}

// Test cases

func TestBuildService_PrepareBuildRequest(t *testing.T) {
	service := createTestBuildService()

	t.Run("success with all fields", func(t *testing.T) {
		gitDir := "/backend"
		lastCommit := "old123"
		template := "FROM node:18"
		installationID := int64(12345)

		container := &dto.BuildContainerInfo{
			ProjectID:           10,
			ContainerID:         1,
			Name:                "test-container",
			Slug:                "test-slug",
			TemplateBody:        &template,
			TemplateConfig:      map[string]interface{}{"PORT": "3000"},
			GitRepositoryURL:    "https://github.com/test/repo",
			GitBranch:           "main",
			GitDirectoryPath:    &gitDir,
			LastBuiltCommitHash: &lastCommit,
			NeedsBuild:          true,
			BuildVars:           map[string]string{"API_KEY": "secret"},
			InstallationID:      &installationID,
		}

		request, err := service.(*builderImpl).prepareBuildRequest(container)

		require.NoError(t, err)
		assert.Equal(t, "test-slug", request.ImageName)
		assert.Equal(t, "https://github.com/test/repo", request.GitHubURL)
		assert.Equal(t, "main", request.GitHubBranch)
		assert.Equal(t, "/backend", request.DirectoryPath)
		assert.Equal(t, "true", request.ForceBuild)
		assert.Equal(t, "old123", request.LastBuildCommitHash)
		assert.Equal(t, "FROM node:18", request.Template)
		assert.Equal(t, "12345", request.InstallationID)

		// Verify JSON fields
		var templateConfig map[string]interface{}
		err = json.Unmarshal(request.DockerfileConfigJSON, &templateConfig)
		require.NoError(t, err)
		assert.Equal(t, "3000", templateConfig["PORT"])

		var buildVars map[string]string
		err = json.Unmarshal(request.BuildEnvJSON, &buildVars)
		require.NoError(t, err)
		assert.Equal(t, "secret", buildVars["API_KEY"])
	})

	t.Run("success with minimal fields", func(t *testing.T) {
		minimalTemplate := "FROM alpine:latest"
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      1,
			Name:             "test-container",
			Slug:             "test-slug",
			TemplateBody:     &minimalTemplate, // Template is required
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       false,
		}

		request, err := service.(*builderImpl).prepareBuildRequest(container)

		require.NoError(t, err)
		assert.Equal(t, "test-slug", request.ImageName)
		assert.Equal(t, "false", request.ForceBuild)
		assert.Equal(t, "", request.DirectoryPath)
		assert.Equal(t, "", request.LastBuildCommitHash)
		assert.Equal(t, "FROM alpine:latest", request.Template)
		assert.Equal(t, "", request.InstallationID)
	})

	t.Run("success with nil template (no template configured)", func(t *testing.T) {
		// Test that nil template is accepted and converted to empty string
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      1,
			Name:             "test-container",
			Slug:             "test-slug",
			TemplateBody:     nil, // Explicitly nil - no template configured
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       false,
		}

		request, err := service.(*builderImpl).prepareBuildRequest(container)

		require.NoError(t, err)
		assert.Equal(t, "test-slug", request.ImageName)
		assert.Equal(t, "", request.Template) // nil should convert to empty string
		assert.Equal(t, "false", request.ForceBuild)
	})

	t.Run("success with empty string template", func(t *testing.T) {
		// Test that empty string template is also accepted
		emptyTemplate := ""
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      1,
			Name:             "test-container",
			Slug:             "test-slug",
			TemplateBody:     &emptyTemplate, // Explicitly empty string
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       false,
		}

		request, err := service.(*builderImpl).prepareBuildRequest(container)

		require.NoError(t, err)
		assert.Equal(t, "test-slug", request.ImageName)
		assert.Equal(t, "", request.Template)
		assert.Equal(t, "false", request.ForceBuild)
	})
}

func TestBuildService_BuildContainer_TriggerFailure(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		findByIDFunc: func(ctx context.Context, id uint) (*build_history.BuildHistory, error) {
			bh := build_history.NewBuildHistory(1)
			bh.SetBuildHistoryID(id)
			return bh, nil
		},
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	tektonBuildClient := &mockTektonBuildClient{
		triggerBuildFunc: func(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
			return nil, projecterrors.ErrTektonBuildFailed
		},
	}

	kubeBuildClient := &mockKubeBuildClient{}

	service := NewBuilder(buildHistoryRepo, tektonBuildClient, kubeBuildClient, testLogger)

	testTemplate := "FROM alpine:latest"
	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
		TemplateBody:     &testTemplate, // Template is required
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		NeedsBuild:       true,
	}

	result, err := service.BuildContainer(ctx, 1, container)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.ErrorMessage, "Failed to trigger Tekton build")
}

func TestBuildService_BuildContainer_FindPipelineRunFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		findByIDFunc: func(ctx context.Context, id uint) (*build_history.BuildHistory, error) {
			bh := build_history.NewBuildHistory(1)
			bh.SetBuildHistoryID(id)
			return bh, nil
		},
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	tektonBuildClient := &mockTektonBuildClient{
		triggerBuildFunc: func(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
			return &dto.TektonBuildResponse{EventID: "test-event-123"}, nil
		},
	}

	kubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	// Use short intervals for test (100ms instead of 10s)
	service := &builderImpl{
		buildHistoryRepo:             buildHistoryRepo,
		tektonBuildClient:            tektonBuildClient,
		kubeBuildClient:              kubeBuildClient,
		logger:                       testLogger,
		pollingInterval:              100 * time.Millisecond,
		findPipelineRunRetryInterval: 100 * time.Millisecond,
	}

	testTemplate := "FROM alpine:latest"
	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
		TemplateBody:     &testTemplate, // Template is required
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		NeedsBuild:       true,
	}

	result, err := service.BuildContainer(ctx, 1, container)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.ErrorMessage, "Failed to find PipelineRun")
}

// TestBuildService_BuildContainer_ContextCancellation tests that context cancellation
// during PipelineRun lookup does NOT mark the build as terminal failure (Option B)
func TestBuildService_BuildContainer_ContextCancellation(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewForTest()

	var savedBuildHistory *build_history.BuildHistory
	buildHistoryRepo := &mockBuildHistoryRepository{
		findByIDFunc: func(ctx context.Context, id uint) (*build_history.BuildHistory, error) {
			bh := build_history.NewBuildHistory(1)
			bh.SetBuildHistoryID(id)
			return bh, nil
		},
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			savedBuildHistory = b
			return nil
		},
	}

	tektonBuildClient := &mockTektonBuildClient{
		triggerBuildFunc: func(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
			return &dto.TektonBuildResponse{EventID: "test-event-123"}, nil
		},
	}

	// Mock: Return context.Canceled on first call
	callCount := 0
	kubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			callCount++
			if callCount == 1 {
				// Simulate context cancellation
				return "", context.Canceled
			}
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	// Use short intervals for test (100ms instead of 10s)
	service := &builderImpl{
		buildHistoryRepo:             buildHistoryRepo,
		tektonBuildClient:            tektonBuildClient,
		kubeBuildClient:              kubeBuildClient,
		logger:                       testLogger,
		pollingInterval:              100 * time.Millisecond,
		findPipelineRunRetryInterval: 100 * time.Millisecond,
		findPipelineRunTotalTimeout:  5 * time.Minute,
		findPipelineRunMaxRetries:    300,
	}

	testTemplate := "FROM alpine:latest"
	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
		TemplateBody:     &testTemplate,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		NeedsBuild:       true,
	}

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := service.BuildContainer(ctx, 1, container)

	// Should return context.Canceled error without marking build as terminal failure
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "Should return context.Canceled error")
	assert.Nil(t, result, "Should not return BuildResult for context cancellation")

	// BuildHistory should NOT be in terminal failure state (Option B)
	// It should remain in untracked state (initial state after creation)
	require.NotNil(t, savedBuildHistory, "BuildHistory should be saved (EventID update)")
	assert.NotEqual(t, build_history.BuildHistoryStatusBackendTrackingFailed, savedBuildHistory.Status(),
		"BuildHistory should NOT be in backend_tracking_failed state on context cancellation")
}

func TestBuildService_HandleBuildSuccess(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	service := &builderImpl{
		buildHistoryRepo: buildHistoryRepo,
		logger:           testLogger,
	}

	buildHistory := build_history.NewBuildHistory(1)
	buildHistory.SetBuildHistoryID(1)

	t.Run("build executed successfully", func(t *testing.T) {
		pipelineRun := &dto.PipelineRun{
			Status:  "True",
			Reason:  "Succeeded",
			Message: "Build completed successfully",
			Results: map[string]string{
				"latest_commit_hash": "abc123def456",
				"image_tag":          "latest",
				"should_build":       "true",
			},
		}

		result, err := service.handleBuildSuccess(ctx, buildHistory, pipelineRun)

		require.NoError(t, err)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "abc123def456", result.LatestCommitHash)
		assert.Equal(t, "latest", result.ImageTag)
		assert.True(t, result.ShouldBuild)
		assert.Equal(t, build_history.BuildHistoryStatusSuccess, buildHistory.Status())
	})

	t.Run("build skipped (no changes)", func(t *testing.T) {
		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(2)

		pipelineRun := &dto.PipelineRun{
			Status:  "True",
			Reason:  "Succeeded",
			Message: "Build skipped (no changes)",
			Results: map[string]string{
				"latest_commit_hash": "abc123def456",
				"image_tag":          "latest",
				"should_build":       "false",
			},
		}

		result, err := service.handleBuildSuccess(ctx, buildHistory, pipelineRun)

		require.NoError(t, err)
		assert.Equal(t, "skipped", result.Status)
		assert.Equal(t, "abc123def456", result.LatestCommitHash)
		assert.False(t, result.ShouldBuild)
		assert.Equal(t, build_history.BuildHistoryStatusSkipped, buildHistory.Status())
	})
}

func TestBuildService_HandleBuildFailure(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	service := &builderImpl{
		buildHistoryRepo: buildHistoryRepo,
		logger:           testLogger,
	}

	t.Run("build failed", func(t *testing.T) {
		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(1)

		pipelineRun := &dto.PipelineRun{
			Status:  "False",
			Reason:  "Failed",
			Message: "Build failed: compilation error",
			Results: map[string]string{},
		}

		result, err := service.handleBuildFailure(ctx, buildHistory, pipelineRun)

		require.NoError(t, err)
		assert.Equal(t, "failed", result.Status)
		assert.Contains(t, result.ErrorMessage, "compilation error")
		assert.Equal(t, build_history.BuildHistoryStatusFailed, buildHistory.Status())
	})

	t.Run("build cancelled", func(t *testing.T) {
		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(2)

		pipelineRun := &dto.PipelineRun{
			Status:  "False",
			Reason:  "Cancelled",
			Message: "Build cancelled by user",
			Results: map[string]string{},
		}

		result, err := service.handleBuildFailure(ctx, buildHistory, pipelineRun)

		require.NoError(t, err)
		assert.Equal(t, "cancelled", result.Status)
		assert.Contains(t, result.ErrorMessage, "cancelled")
		assert.Equal(t, build_history.BuildHistoryStatusCancelled, buildHistory.Status())
	})
}

func TestBuildService_CheckBuildStatus(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	t.Run("status Unknown - still running", func(t *testing.T) {
		kubeBuildClient := &mockKubeBuildClient{
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				return &dto.PipelineRun{
					Status:  "Unknown",
					Reason:  "Running",
					Message: "Build in progress",
				}, nil
			},
		}

		service := &builderImpl{
			buildHistoryRepo: buildHistoryRepo,
			kubeBuildClient:  kubeBuildClient,
			logger:           testLogger,
		}

		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(1)

		result, err := service.checkBuildStatus(ctx, buildHistory, "test-run")

		require.NoError(t, err)
		assert.Nil(t, result) // Still running, no result yet
		assert.Equal(t, build_history.BuildHistoryStatusRunning, buildHistory.Status())
	})

	t.Run("status True - build succeeded", func(t *testing.T) {
		kubeBuildClient := &mockKubeBuildClient{
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				return &dto.PipelineRun{
					Status:  "True",
					Reason:  "Succeeded",
					Message: "Build completed",
					Results: map[string]string{
						"latest_commit_hash": "abc123",
						"image_tag":          "latest",
						"should_build":       "true",
					},
				}, nil
			},
		}

		service := &builderImpl{
			buildHistoryRepo: buildHistoryRepo,
			kubeBuildClient:  kubeBuildClient,
			logger:           testLogger,
		}

		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(1)

		result, err := service.checkBuildStatus(ctx, buildHistory, "test-run")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "abc123", result.LatestCommitHash)
	})

	t.Run("PipelineRun deleted - terminal failure", func(t *testing.T) {
		kubeBuildClient := &mockKubeBuildClient{
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				return nil, projecterrors.ErrKubePipelineRunNotFound
			},
		}

		service := &builderImpl{
			buildHistoryRepo: buildHistoryRepo,
			kubeBuildClient:  kubeBuildClient,
			logger:           testLogger,
		}

		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(1)

		result, err := service.checkBuildStatus(ctx, buildHistory, "test-run")

		// Should return both result and error for terminal state
		require.Error(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "failed", result.Status)
		assert.Contains(t, result.ErrorMessage, "not found in Kubernetes")
		assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingFailed, buildHistory.Status())
	})

	t.Run("GetPipelineRunStatus transient error", func(t *testing.T) {
		kubeBuildClient := &mockKubeBuildClient{
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				return nil, errors.New("network error")
			},
		}

		service := &builderImpl{
			buildHistoryRepo: buildHistoryRepo,
			kubeBuildClient:  kubeBuildClient,
			logger:           testLogger,
		}

		buildHistory := build_history.NewBuildHistory(1)
		buildHistory.SetBuildHistoryID(1)

		result, err := service.checkBuildStatus(ctx, buildHistory, "test-run")

		require.Error(t, err)
		assert.Nil(t, result) // Transient error - no terminal result
		assert.Contains(t, err.Error(), "failed to get PipelineRun status")
		assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingLost, buildHistory.Status())
	})
}

// TestBuildService_FastBuildStartedAt tests that started_at is set for fast builds
// that complete before the first monitoring poll (regression test for Issue #548)
func TestBuildService_FastBuildStartedAt(t *testing.T) {
	t.Parallel()
	t.Run("success - started_at set from PipelineRun.StartTime", func(t *testing.T) {
		t.Parallel()
		testTemplate := "FROM alpine:latest"
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      1,
			Name:             "fast-build",
			Slug:             "fast-build-test",
			TemplateBody:     &testTemplate,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		}

		buildHistory := build_history.NewBuildHistory(container.ContainerID)
		buildHistory.SetBuildHistoryID(1)

		// Mock repository to save BuildHistory
		var savedBuildHistory *build_history.BuildHistory
		mockRepo := &mockBuildHistoryRepository{
			saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
				savedBuildHistory = b
				return nil
			},
		}

		// Mock Tekton client
		eventID := "test-event-123"
		mockTektonClient := &mockTektonBuildClient{
			triggerBuildFunc: func(ctx context.Context, req *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
				return &dto.TektonBuildResponse{EventID: eventID}, nil
			},
		}

		// Mock Kube client - PipelineRun already completed before first poll
		pipelineRunName := "build-run-123"
		startTime := parseTime(t, "2024-01-01T10:00:00Z")
		completionTime := parseTime(t, "2024-01-01T10:01:30Z") // 90 seconds build

		mockKubeClient := &mockKubeBuildClient{
			findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
				return pipelineRunName, nil
			},
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				// Return completed status on first poll
				return &dto.PipelineRun{
					Name:           pipelineRunName,
					Status:         "True",
					Reason:         "Succeeded",
					Message:        "Build completed",
					StartTime:      &startTime,
					CompletionTime: &completionTime,
					Results: map[string]string{
						"latest_commit_hash": "abc123",
						"image_tag":          "abc123",
						"should_build":       "true",
					},
				}, nil
			},
		}

		buildService := &builderImpl{
			buildHistoryRepo:             mockRepo,
			tektonBuildClient:            mockTektonClient,
			kubeBuildClient:              mockKubeClient,
			logger:                       logger.NewForTest(),
			pollingInterval:              100 * time.Millisecond,
			findPipelineRunRetryInterval: 100 * time.Millisecond,
			findPipelineRunTotalTimeout:  5 * time.Minute,
			findPipelineRunMaxRetries:    300,
		}

		// Execute
		ctx := context.Background()
		result, err := buildService.BuildContainer(ctx, buildHistory.BuildHistoryID, container)

		// Verify result
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "abc123", result.LatestCommitHash)
		assert.True(t, result.ShouldBuild)

		// Verify BuildHistory started_at is set
		require.NotNil(t, savedBuildHistory, "BuildHistory should be saved")
		savedStartedAt, hasStartedAt := savedBuildHistory.StartedAt()
		assert.True(t, hasStartedAt, "started_at should be set for fast builds")
		assert.Equal(t, startTime, savedStartedAt, "started_at should match PipelineRun.StartTime")

		// Verify finished_at is also set
		savedFinishedAt, hasFinishedAt := savedBuildHistory.FinishedAt()
		assert.True(t, hasFinishedAt, "finished_at should be set")
		assert.Equal(t, completionTime, savedFinishedAt, "finished_at should match PipelineRun.CompletionTime")
	})

	t.Run("failure - started_at set from PipelineRun.StartTime", func(t *testing.T) {
		t.Parallel()
		testTemplate := "FROM alpine:latest"
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      2,
			Name:             "fast-failed-build",
			Slug:             "fast-failed-build-test",
			TemplateBody:     &testTemplate,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       true,
		}

		buildHistory := build_history.NewBuildHistory(container.ContainerID)
		buildHistory.SetBuildHistoryID(2)

		// Mock repository to save BuildHistory
		var savedBuildHistory *build_history.BuildHistory
		mockRepo := &mockBuildHistoryRepository{
			saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
				savedBuildHistory = b
				return nil
			},
		}

		// Mock Tekton client
		eventID := "test-event-456"
		mockTektonClient := &mockTektonBuildClient{
			triggerBuildFunc: func(ctx context.Context, req *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
				return &dto.TektonBuildResponse{EventID: eventID}, nil
			},
		}

		// Mock Kube client - PipelineRun failed before first poll
		pipelineRunName := "build-run-456"
		startTime := parseTime(t, "2024-01-01T11:00:00Z")
		completionTime := parseTime(t, "2024-01-01T11:00:45Z") // 45 seconds before failure

		mockKubeClient := &mockKubeBuildClient{
			findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
				return pipelineRunName, nil
			},
			getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
				// Return failed status on first poll
				return &dto.PipelineRun{
					Name:           pipelineRunName,
					Status:         "False",
					Reason:         "Failed",
					Message:        "Build failed: compilation error",
					StartTime:      &startTime,
					CompletionTime: &completionTime,
					Results:        map[string]string{},
				}, nil
			},
		}

		buildService := &builderImpl{
			buildHistoryRepo:             mockRepo,
			tektonBuildClient:            mockTektonClient,
			kubeBuildClient:              mockKubeClient,
			logger:                       logger.NewForTest(),
			pollingInterval:              100 * time.Millisecond,
			findPipelineRunRetryInterval: 100 * time.Millisecond,
			findPipelineRunTotalTimeout:  5 * time.Minute,
			findPipelineRunMaxRetries:    300,
		}

		// Execute
		ctx := context.Background()
		result, err := buildService.BuildContainer(ctx, buildHistory.BuildHistoryID, container)

		// Verify result
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "failed", result.Status)

		// Verify BuildHistory started_at is set even for failed builds
		require.NotNil(t, savedBuildHistory, "BuildHistory should be saved")
		savedStartedAt, hasStartedAt := savedBuildHistory.StartedAt()
		assert.True(t, hasStartedAt, "started_at should be set for fast failed builds")
		assert.Equal(t, startTime, savedStartedAt, "started_at should match PipelineRun.StartTime")

		// Verify finished_at is also set
		savedFinishedAt, hasFinishedAt := savedBuildHistory.FinishedAt()
		assert.True(t, hasFinishedAt, "finished_at should be set")
		assert.Equal(t, completionTime, savedFinishedAt, "finished_at should match PipelineRun.CompletionTime")
	})
}

// TestBuildService_InitialCheckTerminalError tests that monitorBuildStatus returns
// immediately when the first checkBuildStatus call returns a terminal result with error
// (regression test for infinite retry bug when PipelineRun is deleted before polling starts)
func TestBuildService_InitialCheckTerminalError(t *testing.T) {
	ctx := context.Background()

	// Mock: PipelineRun deleted before monitoring starts
	mockKubeClient := &mockKubeBuildClient{
		getPipelineRunStatusFunc: func(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
			return nil, projecterrors.ErrKubePipelineRunNotFound
		},
	}

	var savedBuildHistory *build_history.BuildHistory
	mockBuildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, bh *build_history.BuildHistory) error {
			savedBuildHistory = bh
			return nil
		},
	}

	buildService := &builderImpl{
		buildHistoryRepo:             mockBuildHistoryRepo,
		kubeBuildClient:              mockKubeClient,
		logger:                       logger.NewForTest(),
		pollingInterval:              100 * time.Millisecond,
		findPipelineRunRetryInterval: 100 * time.Millisecond,
	}

	buildHistory := build_history.NewBuildHistory(1)
	buildHistory.SetBuildHistoryID(1)
	if err := buildHistory.UpdateRunningStatus(nil, nil); err != nil {
		t.Fatalf("Failed to set running status: %v", err)
	}

	// Call monitorBuildStatus - should return immediately without infinite loop
	result, err := buildService.monitorBuildStatus(ctx, buildHistory, "test-pipeline-run")

	// Should return terminal result with error (not retry infinitely)
	require.Error(t, err, "Should return error for terminal failure")
	require.NotNil(t, result, "Should return terminal result")
	assert.Equal(t, "failed", result.Status, "Status should be failed")
	assert.Contains(t, result.ErrorMessage, "not found in Kubernetes")

	// BuildHistory should be in terminal state
	require.NotNil(t, savedBuildHistory, "BuildHistory should be saved")
	assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingFailed, savedBuildHistory.Status())
}

// Helper functions

func createTestBuildService() Builder {
	return &builderImpl{
		buildHistoryRepo:  &mockBuildHistoryRepository{},
		tektonBuildClient: &mockTektonBuildClient{},
		kubeBuildClient:   &mockKubeBuildClient{},
		logger:            logger.NewForTest(),
	}
}

func parseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	require.NoError(t, err, "Failed to parse time: %s", timeStr)
	return parsedTime
}

// TestBuildService_FindPipelineRunWithRetry_Timeout tests the scenario where
// timeout occurs before PipelineRun is found, ensuring BUILD_HISTORY is updated to terminal state
func TestBuildService_FindPipelineRunWithRetry_Timeout(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedBuildHistory *build_history.BuildHistory
	mockBuildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			savedBuildHistory = b
			return nil
		},
	}

	// Mock: Always return "not found" to trigger timeout
	mockKubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	// Create service with short timeout to test the actual time.After timeout path
	// timeout=47ms, interval=10ms → calculatedRetries=4 (47/10 integer division)
	// maxRetries=100 → min(4, 100)=4, so loop runs 4 times (40ms) then timeout fires at 47ms
	service := &builderImpl{
		buildHistoryRepo:             mockBuildHistoryRepo,
		kubeBuildClient:              mockKubeBuildClient,
		logger:                       testLogger,
		findPipelineRunRetryInterval: 10 * time.Millisecond,  // Fast retries for testing
		findPipelineRunTotalTimeout:  47 * time.Millisecond,  // Timeout fires 7ms after 4th attempt
		findPipelineRunMaxRetries:    100,                    // High enough that calculatedRetries takes precedence
		pollingInterval:              100 * time.Millisecond, // Not used in this test
	}

	buildHistory := build_history.NewBuildHistory(1)
	eventID := "test-event-timeout"

	// Act: Call without context timeout - let the internal time.After fire
	pipelineRunName, err := service.findPipelineRunWithRetry(ctx, buildHistory, eventID)

	// Assert: Should return error (either timeout or retry exhaustion depending on timing)
	require.Error(t, err)
	assert.Empty(t, pipelineRunName)
	// Due to timing, either timeout or retry exhaustion can occur - both are valid terminal states
	assert.True(t,
		strings.Contains(err.Error(), "timeout waiting for PipelineRun") ||
			strings.Contains(err.Error(), "PipelineRun not found after"),
		"error should indicate PipelineRun lookup failure: %v", err)

	// Assert: BUILD_HISTORY should be updated to backend_tracking_failed and saved
	require.NotNil(t, savedBuildHistory, "BUILD_HISTORY should be saved after PipelineRun lookup failure")
	assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingFailed, savedBuildHistory.Status())
}

// TestBuildService_FindPipelineRunWithRetry_RetryExhaustion tests the scenario where
// all retry attempts are exhausted without finding PipelineRun, ensuring BUILD_HISTORY is updated to terminal state
func TestBuildService_FindPipelineRunWithRetry_RetryExhaustion(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedBuildHistory *build_history.BuildHistory
	mockBuildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			savedBuildHistory = b
			return nil
		},
	}

	// Mock: Always return "not found" to exhaust retries
	attemptCount := 0
	mockKubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			attemptCount++
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	// Create service with settings that exhaust retries before timeout
	// timeout=5s, interval=10ms, maxRetries=10 → retry exhaustion fires first
	// 10 attempts × 10ms = 100ms << 5s
	service := &builderImpl{
		buildHistoryRepo:             mockBuildHistoryRepo,
		kubeBuildClient:              mockKubeBuildClient,
		logger:                       testLogger,
		findPipelineRunRetryInterval: 10 * time.Millisecond, // Fast retries for testing
		findPipelineRunTotalTimeout:  5 * time.Second,       // Long timeout, won't fire
		findPipelineRunMaxRetries:    10,                    // Low limit, exhausts first
		pollingInterval:              100 * time.Millisecond,
	}

	buildHistory := build_history.NewBuildHistory(1)
	eventID := "test-event-exhaustion"

	// Act: Call without context timeout - let retry exhaustion fire
	pipelineRunName, err := service.findPipelineRunWithRetry(ctx, buildHistory, eventID)

	// Assert: Should return retry exhaustion error
	require.Error(t, err)
	assert.Empty(t, pipelineRunName)
	assert.Contains(t, err.Error(), "PipelineRun not found after")
	assert.Equal(t, 10, attemptCount, "Should have attempted exactly maxRetries times")

	// Assert: BUILD_HISTORY should be updated to backend_tracking_failed and saved
	require.NotNil(t, savedBuildHistory, "BUILD_HISTORY should be saved after retry exhaustion")
	assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingFailed, savedBuildHistory.Status())
}

// TestBuildService_FindPipelineRunWithRetry_UpdateFailure tests that
// the method continues to return error even if BUILD_HISTORY update fails
func TestBuildService_FindPipelineRunWithRetry_UpdateFailure(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	// Mock: Save always fails
	mockBuildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return errors.New("database error")
		},
	}

	// Mock: Always return "not found"
	mockKubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	service := &builderImpl{
		buildHistoryRepo:             mockBuildHistoryRepo,
		kubeBuildClient:              mockKubeBuildClient,
		logger:                       testLogger,
		findPipelineRunRetryInterval: 50 * time.Millisecond,
		pollingInterval:              100 * time.Millisecond,
	}

	buildHistory := build_history.NewBuildHistory(1)
	eventID := "test-event-update-failure"

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	// Act
	pipelineRunName, err := service.findPipelineRunWithRetry(ctxWithTimeout, buildHistory, eventID)

	// Assert: Should still return the original error even if update fails
	assert.Error(t, err)
	assert.Empty(t, pipelineRunName)
	// The update/save failure should be logged but not prevent error return
}

// TestBuildService_BuildContainer_SaveRetryAfterFindPipelineRunFailure tests that
// BuildContainer retries Save when findPipelineRunWithRetry fails to persist
func TestBuildService_BuildContainer_SaveRetryAfterFindPipelineRunFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	testLogger := logger.NewForTest()

	saveCallCount := 0
	var savedBuildHistory *build_history.BuildHistory

	// Mock: First Save of backend_tracking_failed fails (in findPipelineRunWithRetry),
	// second Save of backend_tracking_failed succeeds (in BuildContainer)
	buildHistoryRepo := &mockBuildHistoryRepository{
		findByIDFunc: func(ctx context.Context, id uint) (*build_history.BuildHistory, error) {
			bh := build_history.NewBuildHistory(1)
			bh.SetBuildHistoryID(id)
			return bh, nil
		},
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			// Only track/fail saves when BuildHistory is in backend_tracking_failed state
			if b.Status() == build_history.BuildHistoryStatusBackendTrackingFailed {
				saveCallCount++
				if saveCallCount == 1 {
					// First save attempt of backend_tracking_failed (from findPipelineRunWithRetry) fails
					return errors.New("database connection error")
				}
				// Second save attempt of backend_tracking_failed (from BuildContainer) succeeds
				savedBuildHistory = b
			}
			// All other saves succeed
			return nil
		},
	}

	tektonBuildClient := &mockTektonBuildClient{
		triggerBuildFunc: func(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
			return &dto.TektonBuildResponse{EventID: "test-event-123"}, nil
		},
	}

	kubeBuildClient := &mockKubeBuildClient{
		findPipelineRunNameByEventIDFunc: func(ctx context.Context, eventID string) (string, error) {
			return "", projecterrors.ErrKubePipelineRunNotFound
		},
	}

	service := &builderImpl{
		buildHistoryRepo:             buildHistoryRepo,
		tektonBuildClient:            tektonBuildClient,
		kubeBuildClient:              kubeBuildClient,
		logger:                       testLogger,
		pollingInterval:              100 * time.Millisecond,
		findPipelineRunRetryInterval: 10 * time.Millisecond, // Fast retries for testing
		findPipelineRunTotalTimeout:  5 * time.Second,       // Long timeout, won't fire
		findPipelineRunMaxRetries:    10,                    // Low limit, exhausts quickly
	}

	testTemplate := "FROM alpine:latest"
	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
		TemplateBody:     &testTemplate,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		NeedsBuild:       true,
	}

	result, err := service.BuildContainer(ctx, 1, container)

	// Assert: BuildContainer returns error (from findPipelineRunWithRetry failure)
	// but the critical point is that BUILD_HISTORY was saved to DB (via retry in BuildContainer)
	require.Error(t, err)
	// Error can be either timeout or retry exhaustion depending on timing
	assert.True(t,
		strings.Contains(err.Error(), "timeout waiting for PipelineRun") ||
			strings.Contains(err.Error(), "PipelineRun not found after"),
		"error should indicate PipelineRun lookup failure")
	assert.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.ErrorMessage, "Failed to find PipelineRun")

	// Verify Save was called twice for backend_tracking_failed state:
	// - First call from findPipelineRunWithRetry: FAILED (database error)
	// - Second call from BuildContainer: SUCCEEDED (retry)
	assert.Equal(t, 2, saveCallCount, "Save should be called twice for backend_tracking_failed state")

	// Verify BUILD_HISTORY was eventually saved to DB in terminal state (this is the fix!)
	// Without the fix, savedBuildHistory would be nil (DB would remain in 'untracked' state)
	require.NotNil(t, savedBuildHistory, "BuildHistory should be saved to DB via retry in BuildContainer")
	assert.Equal(t, build_history.BuildHistoryStatusBackendTrackingFailed, savedBuildHistory.Status())
	assert.True(t, savedBuildHistory.IsCompleted(), "BuildHistory should be in terminal state")
}
