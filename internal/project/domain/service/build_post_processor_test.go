package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	containervalue "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// Test mock implementations

type mockPostProcessorContainerRepo struct {
	findByIDForUpdateFunc func(ctx context.Context, containerID uint) (*containermodel.Container, error)
	saveFunc              func(ctx context.Context, container *containermodel.Container) error
}

func (m *mockPostProcessorContainerRepo) FindByIDForUpdate(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	if m.findByIDForUpdateFunc != nil {
		return m.findByIDForUpdateFunc(ctx, containerID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPostProcessorContainerRepo) Save(ctx context.Context, container *containermodel.Container) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, container)
	}
	return nil
}

// Implement other required methods as no-op
func (m *mockPostProcessorContainerRepo) Create(ctx context.Context, container *containermodel.Container) error {
	return nil
}
func (m *mockPostProcessorContainerRepo) FindByID(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	return nil, nil
}
func (m *mockPostProcessorContainerRepo) FindByProjectID(ctx context.Context, projectID uint) ([]*containermodel.Container, error) {
	return nil, nil
}
func (m *mockPostProcessorContainerRepo) FindBySlug(ctx context.Context, slug string) (*containermodel.Container, error) {
	return nil, nil
}
func (m *mockPostProcessorContainerRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}
func (m *mockPostProcessorContainerRepo) ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error) {
	return false, nil
}
func (m *mockPostProcessorContainerRepo) Delete(ctx context.Context, containerID uint) error {
	return nil
}
func (m *mockPostProcessorContainerRepo) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return nil
}
func (m *mockPostProcessorContainerRepo) List(ctx context.Context, offset, limit int) ([]*containermodel.Container, error) {
	return nil, nil
}
func (m *mockPostProcessorContainerRepo) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockPostProcessorContainerRepo) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	return 0, nil
}
func (m *mockPostProcessorContainerRepo) CountByTemplateID(ctx context.Context, templateID uint) (int64, error) {
	return 0, nil
}
func (m *mockPostProcessorContainerRepo) GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (totalCPU uint32, totalMemory uint32, err error) {
	return 0, 0, nil
}

type mockPostProcessorTxManager struct {
	runInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockPostProcessorTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.runInTxFunc != nil {
		return m.runInTxFunc(ctx, fn)
	}
	// Default: just execute the function without transaction
	return fn(ctx)
}

// Test cases

func TestBuildPostProcessor_UpdateContainerAfterBuild_NonSuccessStatus(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	mockRepo := &mockPostProcessorContainerRepo{}
	mockTxMgr := &mockPostProcessorTxManager{}

	processor := NewBuildPostProcessor(mockRepo, mockTxMgr, testLogger)

	// Test data
	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "failed",
		ErrorMessage:     "build failed",
		LatestCommitHash: "",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	// Execute
	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Verify - should not return error for non-success builds
	assert.NoError(t, err)
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_FetchFails(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	fetchErr := errors.New("database error")
	mockRepo := &mockPostProcessorContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
			return nil, fetchErr
		},
	}
	mockTxMgr := &mockPostProcessorTxManager{}

	processor := NewBuildPostProcessor(mockRepo, mockTxMgr, testLogger)

	// Test data
	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "success",
		LatestCommitHash: "abc123",
		ImageTag:         "latest",
		ShouldBuild:      true,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
	}

	// Execute
	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update container after build")
}

func TestBuildPostProcessor_HasBuildParametersChanged(t *testing.T) {
	// Setup
	testLogger := logger.NewForTest()

	processor := NewBuildPostProcessor(nil, nil, testLogger).(*buildPostProcessorImpl)

	// Create test container
	slug, _ := containervalue.NewContainerSlug("test-container")
	gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpu := uint32(1000)
	mem := uint32(512)
	resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

	container, _ := containermodel.NewContainer(
		1,      // projectID
		"test", // name
		slug,
		gitConfig,
		resourceLimits,
		nil, // templateID
		nil, // templateConfig
		nil, // githubInstallationID
	)

	t.Run("No changes - should return false", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			ContainerID:      1,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateBody:     nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Git URL changed - should return true", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			ContainerID:      1,
			GitRepositoryURL: "https://github.com/test/different-repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateBody:     nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Git branch changed - should return true", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			ContainerID:      1,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "develop",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateBody:     nil,
			TemplateConfig:   nil,
			BuildVars:        nil,
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Template ID changed - should return true", func(t *testing.T) {
		templateID := uint(999)
		snapshot := &dto.BuildContainerInfo{
			ContainerID:      1,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       &templateID, // Different template ID
			TemplateBody:     nil,
			TemplateConfig:   nil,
			BuildVars:        nil,
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("BuildVars changed - should return true", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			ContainerID:      1,
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateBody:     nil,
			TemplateConfig:   nil,
			BuildVars: map[string]string{
				"NEW_VAR": "new_value",
			},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_EmptyCommitHash(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedContainer *containermodel.Container
	mockRepo := &mockPostProcessorContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
			// Create container with existing commit hash
			slug, _ := containervalue.NewContainerSlug("test-container")
			gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
			cpu := uint32(1000)
			mem := uint32(512)
			resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

			container, _ := containermodel.NewContainer(
				1,
				"test",
				slug,
				gitConfig,
				resourceLimits,
				nil,
				nil,
				nil,
			)

			// Set previous commit hash
			previousHash := "previous-commit-123"
			container.SetLastBuiltCommitHash(&previousHash)

			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedContainer = container
			return nil
		},
	}
	mockTxMgr := &mockPostProcessorTxManager{}

	processor := NewBuildPostProcessor(mockRepo, mockTxMgr, testLogger)

	// Test data - build result with empty commit hash
	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "success",
		LatestCommitHash: "", // Empty hash (e.g., skipped build)
		ImageTag:         "latest",
		ShouldBuild:      false,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		TemplateID:       nil,
		BuildVars:        map[string]string{},
	}

	// Execute
	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, savedContainer)

	// Previous commit hash should be preserved (not overwritten with empty string)
	commitHash := savedContainer.LastBuiltGitCommitHash()
	assert.NotNil(t, commitHash)
	assert.Equal(t, "previous-commit-123", *commitHash)

	// needs_build should still be cleared
	assert.False(t, savedContainer.NeedsBuild())
}

func TestBuildPostProcessor_BuildVarsComparison(t *testing.T) {
	testLogger := logger.NewForTest()
	processor := NewBuildPostProcessor(nil, nil, testLogger).(*buildPostProcessorImpl)

	// Create container with build vars
	slug, _ := containervalue.NewContainerSlug("test-container")
	gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpu := uint32(1000)
	mem := uint32(512)
	resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

	container, _ := containermodel.NewContainer(
		1,
		"test",
		slug,
		gitConfig,
		resourceLimits,
		nil,
		nil,
		nil,
	)
	container.SetContainerID(1) // Required for AddBuildVar

	// Add build vars to container
	_, _ = container.AddBuildVar("VAR1", "value1")
	_, _ = container.AddBuildVar("VAR2", "value2")

	t.Run("Identical build vars - no change detected", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Different build var values - change detected", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars: map[string]string{
				"VAR1": "different_value",
				"VAR2": "value2",
			},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Additional build var - change detected", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Missing build var - change detected", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars: map[string]string{
				"VAR1": "value1",
			},
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}

func TestBuildPostProcessor_UpdateContainerAfterBuild_SkippedBuild(t *testing.T) {
	// Setup
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedContainer *containermodel.Container
	previousHash := "previous-commit-456"

	mockRepo := &mockPostProcessorContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
			// Create container with existing commit hash
			slug, _ := containervalue.NewContainerSlug("test-container")
			gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
			cpu := uint32(1000)
			mem := uint32(512)
			resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

			container, _ := containermodel.NewContainer(
				1,
				"test",
				slug,
				gitConfig,
				resourceLimits,
				nil,
				nil,
				nil,
			)

			// Set previous commit hash and needs_build=true
			container.SetLastBuiltCommitHash(&previousHash)
			container.MarkNeedsBuild()

			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedContainer = container
			return nil
		},
	}
	mockTxMgr := &mockPostProcessorTxManager{}

	processor := NewBuildPostProcessor(mockRepo, mockTxMgr, testLogger)

	// Test data - skipped build result
	buildResult := &BuildResult{
		BuildHistoryID:   1,
		Status:           "skipped", // Skipped build (should_build=false)
		LatestCommitHash: "",        // No new commit hash for skipped builds
		ImageTag:         "latest",
		ShouldBuild:      false,
	}

	snapshot := &dto.BuildContainerInfo{
		ContainerID:      10,
		GitRepositoryURL: "https://github.com/test/repo",
		GitBranch:        "main",
		TemplateID:       nil,
		BuildVars:        map[string]string{},
	}

	// Execute
	err := processor.UpdateContainerAfterBuild(ctx, 10, buildResult, snapshot)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, savedContainer)

	// Previous commit hash should be preserved (not overwritten)
	commitHash := savedContainer.LastBuiltGitCommitHash()
	assert.NotNil(t, commitHash)
	assert.Equal(t, previousHash, *commitHash)

	// needs_build should be cleared even for skipped builds
	assert.False(t, savedContainer.NeedsBuild())
}

func TestBuildPostProcessor_InstallationIDChanged(t *testing.T) {
	// Setup
	testLogger := logger.NewForTest()

	processor := NewBuildPostProcessor(nil, nil, testLogger).(*buildPostProcessorImpl)

	// Create test container with installation ID
	slug, _ := containervalue.NewContainerSlug("test-container")
	gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpu := uint32(1000)
	mem := uint32(512)
	resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

	installationID := int64(12345678)
	container, _ := containermodel.NewContainer(
		1,
		"test",
		slug,
		gitConfig,
		resourceLimits,
		nil,
		nil,
		&installationID,
	)

	t.Run("No change - identical installation IDs", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   &installationID,
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Change detected - different installation ID", func(t *testing.T) {
		differentID := int64(87654321)
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   &differentID,
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Change detected - installation ID added", func(t *testing.T) {
		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   nil, // Snapshot has no installation ID
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("Change detected - installation ID removed", func(t *testing.T) {
		// Create container without installation ID
		containerNoID, _ := containermodel.NewContainer(
			1,
			"test",
			slug,
			gitConfig,
			resourceLimits,
			nil,
			nil,
			nil, // No installation ID
		)

		snapshot := &dto.BuildContainerInfo{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   &installationID, // Snapshot has installation ID
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, containerNoID)
		assert.True(t, hasChanged)
	})
}
