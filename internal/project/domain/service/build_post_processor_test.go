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
			TemplateBody:     nil,
			TemplateConfig:   nil,
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
			TemplateBody:     nil,
			TemplateConfig:   nil,
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
			TemplateBody:     nil,
			TemplateConfig:   nil,
		}

		hasChanged := processor.hasBuildParametersChanged(snapshot, container)
		assert.True(t, hasChanged)
	})
}
