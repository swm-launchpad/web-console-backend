package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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

		request, err := service.(*buildServiceImpl).prepareBuildRequest(container)

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
		container := &dto.BuildContainerInfo{
			ProjectID:        10,
			ContainerID:      1,
			Name:             "test-container",
			Slug:             "test-slug",
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			NeedsBuild:       false,
		}

		request, err := service.(*buildServiceImpl).prepareBuildRequest(container)

		require.NoError(t, err)
		assert.Equal(t, "test-slug", request.ImageName)
		assert.Equal(t, "false", request.ForceBuild)
		assert.Equal(t, "", request.DirectoryPath)
		assert.Equal(t, "", request.LastBuildCommitHash)
		assert.Equal(t, "", request.Template)
		assert.Equal(t, "", request.InstallationID)
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

	service := NewBuildService(buildHistoryRepo, tektonBuildClient, kubeBuildClient, testLogger)

	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
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

	service := NewBuildService(buildHistoryRepo, tektonBuildClient, kubeBuildClient, testLogger)

	container := &dto.BuildContainerInfo{
		ProjectID:        10,
		ContainerID:      1,
		Name:             "test-container",
		Slug:             "test-slug",
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

func TestBuildService_HandleBuildSuccess(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	buildHistoryRepo := &mockBuildHistoryRepository{
		saveFunc: func(ctx context.Context, b *build_history.BuildHistory) error {
			return nil
		},
	}

	service := &buildServiceImpl{
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

	service := &buildServiceImpl{
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

		service := &buildServiceImpl{
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

		service := &buildServiceImpl{
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

		service := &buildServiceImpl{
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

		service := &buildServiceImpl{
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

// Helper function

func createTestBuildService() BuildService {
	return &buildServiceImpl{
		buildHistoryRepo:  &mockBuildHistoryRepository{},
		tektonBuildClient: &mockTektonBuildClient{},
		kubeBuildClient:   &mockKubeBuildClient{},
		logger:            logger.NewForTest(),
	}
}
