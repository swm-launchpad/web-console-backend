package build

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	containervalue "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// Mock implementations for testing

type mockUpdateContainerRepo struct {
	findByIDForUpdateFunc func(ctx context.Context, containerID uint) (*containermodel.Container, error)
	saveFunc              func(ctx context.Context, container *containermodel.Container) error
}

func (m *mockUpdateContainerRepo) FindByIDForUpdate(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	if m.findByIDForUpdateFunc != nil {
		return m.findByIDForUpdateFunc(ctx, containerID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUpdateContainerRepo) Save(ctx context.Context, container *containermodel.Container) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, container)
	}
	return nil
}

// Implement other required methods as no-op
func (m *mockUpdateContainerRepo) Create(ctx context.Context, container *containermodel.Container) error {
	return nil
}
func (m *mockUpdateContainerRepo) FindByID(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	return nil, nil
}
func (m *mockUpdateContainerRepo) FindByProjectID(ctx context.Context, projectID uint) ([]*containermodel.Container, error) {
	return nil, nil
}
func (m *mockUpdateContainerRepo) FindBySlug(ctx context.Context, slug string) (*containermodel.Container, error) {
	return nil, nil
}
func (m *mockUpdateContainerRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) Delete(ctx context.Context, containerID uint) error {
	return nil
}
func (m *mockUpdateContainerRepo) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return nil
}
func (m *mockUpdateContainerRepo) List(ctx context.Context, offset, limit int) ([]*containermodel.Container, error) {
	return nil, nil
}
func (m *mockUpdateContainerRepo) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockUpdateContainerRepo) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	return 0, nil
}
func (m *mockUpdateContainerRepo) CountByTemplateID(ctx context.Context, templateID uint) (int64, error) {
	return 0, nil
}
func (m *mockUpdateContainerRepo) GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (totalCPU uint32, totalMemory uint32, err error) {
	return 0, 0, nil
}
func (m *mockUpdateContainerRepo) CheckInternalPortExistsInProject(ctx context.Context, projectID uint, internalPort uint16) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) FindAllSlugsByProjectIDIncludingDeleted(ctx context.Context, projectID uint) ([]string, error) {
	return nil, nil
}
func (m *mockUpdateContainerRepo) CheckFQDNExists(ctx context.Context, fqdn string) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) CheckFQDNExistsForProject(ctx context.Context, fqdn string, projectID uint) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) CheckFQDNExistsForProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) CheckInternalPortExistsInProjectExcludingSelf(ctx context.Context, projectID uint, internalPort uint16, networkID uint) (bool, error) {
	return false, nil
}
func (m *mockUpdateContainerRepo) SoftDeleteNetworksByContainerID(ctx context.Context, containerID uint) error {
	return nil
}
func (m *mockUpdateContainerRepo) SoftDeleteNetworkByID(ctx context.Context, networkID uint) error {
	return nil
}
func (m *mockUpdateContainerRepo) UpdateNetworkTektonEventID(ctx context.Context, networkID uint, tektonEventID string) error {
	return nil
}
func (m *mockUpdateContainerRepo) UpdateNetworkNodePortResult(ctx context.Context, networkID uint, externalIP string, externalPort uint16, expiresAt time.Time) error {
	return nil
}

func (m *mockUpdateContainerRepo) FindContainerByWebhookToken(ctx context.Context, token string) (containerID uint32, projectID uint32, err error) {
	return 0, 0, nil
}

func (m *mockUpdateContainerRepo) ExistsWebhookToken(ctx context.Context, token string) (bool, error) {
	return false, nil
}

type mockUpdateTxManager struct {
	runInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockUpdateTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.runInTxFunc != nil {
		return m.runInTxFunc(ctx, fn)
	}
	// Default: just execute the function without transaction
	return fn(ctx)
}

// Test cases

func TestUpdateContainerAfterBuildUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedContainer *containermodel.Container
	mockRepo := &mockUpdateContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
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
			container.MarkNeedsBuild()
			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedContainer = container
			return nil
		},
	}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	commitHash := "abc123"
	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "success",
		CommitHash:  commitHash,
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   nil,
		},
	}

	wasUpdated, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.True(t, wasUpdated, "container should be updated for successful build")
	require.NotNil(t, savedContainer)
	assert.False(t, savedContainer.NeedsBuild())
	assert.Equal(t, commitHash, *savedContainer.LastBuiltGitCommitHash())
}

func TestUpdateContainerAfterBuildUseCase_Execute_FailedBuild(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	mockRepo := &mockUpdateContainerRepo{}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "failed",
		CommitHash:  "",
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
		},
	}

	wasUpdated, err := useCase.Execute(ctx, input)

	// Should not return error, just skip update
	assert.NoError(t, err)
	assert.False(t, wasUpdated, "container should not be updated for failed build")
}

func TestUpdateContainerAfterBuildUseCase_Execute_ParametersChanged(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedCalled bool
	mockRepo := &mockUpdateContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
			slug, _ := containervalue.NewContainerSlug("test-container")
			// Current state has different branch
			gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "develop", nil)
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
			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedCalled = true
			return nil
		},
	}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "success",
		CommitHash:  "abc123",
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main", // Snapshot has "main", current has "develop"
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   nil,
		},
	}

	wasUpdated, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.False(t, wasUpdated, "container should not be updated when parameters changed")
	assert.False(t, savedCalled, "Save should not be called when parameters changed")
}

func TestUpdateContainerAfterBuildUseCase_Execute_SkippedBuild(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedContainer *containermodel.Container
	previousHash := "old-commit"

	mockRepo := &mockUpdateContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
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
			container.SetLastBuiltCommitHash(&previousHash)
			container.MarkNeedsBuild()
			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedContainer = container
			return nil
		},
	}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "skipped",
		CommitHash:  "", // Skipped builds don't have new commit
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   nil,
		},
	}

	wasUpdated, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.True(t, wasUpdated, "container should be updated for skipped build to clear needs_build flag")
	require.NotNil(t, savedContainer)
	assert.False(t, savedContainer.NeedsBuild())
	// Commit hash should be preserved for skipped builds
	assert.Equal(t, previousHash, *savedContainer.LastBuiltGitCommitHash())
}

func TestUpdateContainerAfterBuildUseCase_Execute_EmptyCommitHash(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	var savedContainer *containermodel.Container
	previousHash := "previous-commit"

	mockRepo := &mockUpdateContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
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
			container.SetLastBuiltCommitHash(&previousHash)
			container.MarkNeedsBuild()
			return container, nil
		},
		saveFunc: func(ctx context.Context, container *containermodel.Container) error {
			savedContainer = container
			return nil
		},
	}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "success",
		CommitHash:  "", // Empty commit hash (for containers without GitHub URL)
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			GitDirectoryPath: nil,
			TemplateID:       nil,
			TemplateConfig:   nil,
			BuildVars:        map[string]string{},
			InstallationID:   nil,
		},
	}

	wasUpdated, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.True(t, wasUpdated, "container should be updated even with empty commit hash")
	require.NotNil(t, savedContainer)
	assert.False(t, savedContainer.NeedsBuild())
	// Empty commit hash should be replaced with "latest" tag
	assert.Equal(t, "latest", *savedContainer.LastBuiltGitCommitHash())
}

func TestUpdateContainerAfterBuildUseCase_Execute_RepositoryError(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewForTest()

	repoErr := errors.New("database error")
	mockRepo := &mockUpdateContainerRepo{
		findByIDForUpdateFunc: func(ctx context.Context, containerID uint) (*containermodel.Container, error) {
			return nil, repoErr
		},
	}
	mockTxMgr := &mockUpdateTxManager{}

	useCase := NewUpdateContainerAfterBuildUseCase(mockRepo, mockTxMgr, testLogger)

	input := UpdateContainerAfterBuildInput{
		ContainerID: 10,
		BuildStatus: "success",
		CommitHash:  "abc123",
		SnapshotBefore: &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
		},
	}

	_, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update container after build")
}

func TestUpdateContainerAfterBuildUseCase_BuildVarsComparison(t *testing.T) {
	testLogger := logger.NewForTest()
	useCase := NewUpdateContainerAfterBuildUseCase(nil, nil, testLogger)

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
	container.SetContainerID(1)
	_, _ = container.AddBuildVar("VAR1", "value1")
	_, _ = container.AddBuildVar("VAR2", "value2")

	t.Run("Identical build vars", func(t *testing.T) {
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			BuildVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Different build var value", func(t *testing.T) {
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			BuildVars: map[string]string{
				"VAR1": "different_value",
				"VAR2": "value2",
			},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}

func TestUpdateContainerAfterBuildUseCase_TemplateConfigComparison(t *testing.T) {
	testLogger := logger.NewForTest()
	useCase := NewUpdateContainerAfterBuildUseCase(nil, nil, testLogger)

	slug, _ := containervalue.NewContainerSlug("test-container")
	gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpu := uint32(1000)
	mem := uint32(512)
	resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

	templateConfig := map[string]interface{}{
		"port":    float64(8080),
		"command": "npm start",
	}

	container, _ := containermodel.NewContainer(
		1,
		"test",
		slug,
		gitConfig,
		resourceLimits,
		nil,
		templateConfig,
		nil,
	)

	t.Run("Identical template config", func(t *testing.T) {
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			TemplateConfig: map[string]interface{}{
				"port":    float64(8080),
				"command": "npm start",
			},
			BuildVars: map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Different template config value", func(t *testing.T) {
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			TemplateConfig: map[string]interface{}{
				"port":    float64(3000), // Different port
				"command": "npm start",
			},
			BuildVars: map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}

func TestUpdateContainerAfterBuildUseCase_InstallationIDComparison(t *testing.T) {
	testLogger := logger.NewForTest()
	useCase := NewUpdateContainerAfterBuildUseCase(nil, nil, testLogger)

	slug, _ := containervalue.NewContainerSlug("test-container")
	gitConfig, _ := containervalue.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpu := uint32(1000)
	mem := uint32(512)
	resourceLimits, _ := containervalue.NewResourceLimits(&cpu, &mem)

	t.Run("Both nil - no change", func(t *testing.T) {
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

		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			InstallationID:   nil,
			BuildVars:        map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Same installation ID - no change", func(t *testing.T) {
		installationID := int64(12345)
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

		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			InstallationID:   &installationID,
			BuildVars:        map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.False(t, hasChanged)
	})

	t.Run("Different installation ID - change detected", func(t *testing.T) {
		currentID := int64(12345)
		container, _ := containermodel.NewContainer(
			1,
			"test",
			slug,
			gitConfig,
			resourceLimits,
			nil,
			nil,
			&currentID,
		)

		snapshotID := int64(67890)
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			InstallationID:   &snapshotID,
			BuildVars:        map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("nil to non-nil - change detected", func(t *testing.T) {
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

		snapshotID := int64(12345)
		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			InstallationID:   &snapshotID,
			BuildVars:        map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})

	t.Run("non-nil to nil - change detected", func(t *testing.T) {
		currentID := int64(12345)
		container, _ := containermodel.NewContainer(
			1,
			"test",
			slug,
			gitConfig,
			resourceLimits,
			nil,
			nil,
			&currentID,
		)

		snapshot := &BuildParametersSnapshot{
			GitRepositoryURL: "https://github.com/test/repo",
			GitBranch:        "main",
			InstallationID:   nil,
			BuildVars:        map[string]string{},
		}

		hasChanged := useCase.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}
