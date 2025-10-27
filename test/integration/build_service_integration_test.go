package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// mockBuildHistoryRepository is an in-memory implementation of BuildHistoryRepository
// for integration testing. It stores build histories in a map.
type mockBuildHistoryRepository struct {
	buildHistories map[uint]*build_history.BuildHistory
	nextID         uint
}

func newMockBuildHistoryRepository() *mockBuildHistoryRepository {
	return &mockBuildHistoryRepository{
		buildHistories: make(map[uint]*build_history.BuildHistory),
		nextID:         1,
	}
}

func (m *mockBuildHistoryRepository) Create(ctx context.Context, b *build_history.BuildHistory) error {
	b.SetBuildHistoryID(m.nextID)
	m.buildHistories[m.nextID] = b
	m.nextID++
	return nil
}

func (m *mockBuildHistoryRepository) Save(ctx context.Context, b *build_history.BuildHistory) error {
	m.buildHistories[b.BuildHistoryID] = b
	return nil
}

func (m *mockBuildHistoryRepository) FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error) {
	if bh, ok := m.buildHistories[buildHistoryID]; ok {
		return bh, nil
	}
	return nil, projecterrors.ErrBuildHistoryNotFound
}

func (m *mockBuildHistoryRepository) FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error) {
	return nil, projecterrors.ErrBuildHistoryNotFound
}

func (m *mockBuildHistoryRepository) FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

func (m *mockBuildHistoryRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error) {
	return nil, projecterrors.ErrBuildHistoryNotFound
}

func (m *mockBuildHistoryRepository) FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error) {
	return nil, nil
}

// setupBuildServiceIntegrationTest performs common setup for build service integration tests.
// Returns context and performs environment validation.
func setupBuildServiceIntegrationTest(t *testing.T) context.Context {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env.test
	helper.LoadTestEnv(t)

	// Verify required environment variables for build
	requiredEnvVars := []string{
		"TEKTON_BUILD_URL",
		"TEKTON_API_AUTH",
		"KUBE_API_SERVER",
		"KUBE_SERVICE_ACCOUNT_TOKEN",
		"KUBE_BUILD_NAMESPACE",
		"KUBE_CA_CERT_PATH",
	}

	for _, envVar := range requiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			t.Skipf("Skipping test: required environment variable %s is not set", envVar)
		}
	}

	return context.Background()
}

// TestBuildServiceIntegration_SpringHelloWorld tests BuildService with a Spring Boot Maven project.
// This test requires actual Tekton and Kubernetes infrastructure to be available.
//
// Prerequisites:
// - Tekton EventListener for builds must be accessible
// - Kubernetes API server must be accessible
// - Environment variables must be set in .env.test
func TestBuildServiceIntegration_SpringHelloWorld(t *testing.T) {
	t.Parallel() // Enable parallel execution for long-running build test

	ctx := setupBuildServiceIntegrationTest(t)

	// Test configuration
	githubURL := "https://github.com/paulczar/spring-helloworld.git"
	githubBranch := "master"
	imageName := "spring-helloworld-buildservice-test-01"
	directoryPath := "."
	templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/springboot-maven.dockerfile.tmpl"

	// Read template
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Skipf("Skipping test: template file not found: %v", err)
	}
	templateContent := string(templateBytes)

	// Template configuration
	templateConfig := map[string]interface{}{
		"maven_version": "3.6",
		"java_version":  "11",
		"app_port":      "8080",
	}

	// Build environment variables
	buildVars := map[string]string{}

	// Create BuildContainerInfo
	container := &dto.BuildContainerInfo{
		ProjectID:        11, // Test ID range: 1-100
		ContainerID:      1,
		Name:             "spring-hello-world",
		Slug:             imageName,
		TemplateBody:     &templateContent,
		TemplateConfig:   templateConfig,
		GitRepositoryURL: githubURL,
		GitBranch:        githubBranch,
		GitDirectoryPath: &directoryPath,
		NeedsBuild:       true,
		BuildVars:        buildVars,
	}

	// Create mock repository
	mockRepo := newMockBuildHistoryRepository()

	// Create BuildHistory record
	bh := build_history.NewBuildHistory(container.ContainerID)
	err = mockRepo.Create(ctx, bh)
	require.NoError(t, err, "Failed to create BuildHistory")

	buildHistoryID := bh.BuildHistoryID
	t.Logf("Created BuildHistory with ID: %d", buildHistoryID)

	// Create real TektonBuildClient
	tektonClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create TektonBuildClient: %v", err)
	}

	// Create real KubeBuildClient
	kubeClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create KubeBuildClient: %v", err)
	}

	// Create BuildService with real clients and mock repository
	buildService := service.NewBuildService(
		mockRepo,
		tektonClient,
		kubeClient,
		logger.NewForTest(),
	)

	// Execute build (this will take several minutes)
	t.Logf("Starting build for container: %s", container.Name)
	startTime := time.Now()

	result, err := buildService.BuildContainer(ctx, buildHistoryID, container)

	elapsed := time.Since(startTime)
	t.Logf("Build completed in %v", elapsed)

	// Verify result
	require.NoError(t, err, "BuildContainer should not return error")
	require.NotNil(t, result, "BuildResult should not be nil")

	assert.Equal(t, buildHistoryID, result.BuildHistoryID, "BuildHistoryID should match")
	assert.Equal(t, "success", result.Status, "Build should succeed")
	assert.True(t, result.ShouldBuild, "ShouldBuild should be true")
	assert.NotEmpty(t, result.LatestCommitHash, "LatestCommitHash should be set")
	assert.NotEmpty(t, result.ImageTag, "ImageTag should be set")
	assert.Empty(t, result.ErrorMessage, "ErrorMessage should be empty")

	t.Logf("Build Result: Status=%s, CommitHash=%s, ImageTag=%s",
		result.Status, result.LatestCommitHash, result.ImageTag)

	// Verify BuildHistory was updated
	updatedBH, err := mockRepo.FindByID(ctx, buildHistoryID)
	require.NoError(t, err, "Should find updated BuildHistory")
	assert.Equal(t, build_history.BuildHistoryStatusSuccess, updatedBH.Status(), "BuildHistory status should be success")
}

// TestBuildServiceIntegration_MySQL tests BuildService with MySQL custom build (no GitHub).
// This test requires actual Tekton and Kubernetes infrastructure to be available.
//
// Prerequisites:
// - Tekton EventListener for builds must be accessible
// - Kubernetes API server must be accessible
// - Environment variables must be set in .env.test
func TestBuildServiceIntegration_MySQL(t *testing.T) {
	t.Parallel() // Enable parallel execution for long-running build test

	ctx := setupBuildServiceIntegrationTest(t)

	// Test configuration
	imageName := "mysql-custom-buildservice-test-01"
	templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/mysql.dockerfile.tmpl"

	// Read template
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Skipf("Skipping test: template file not found: %v", err)
	}
	templateContent := string(templateBytes)

	// Template configuration for MySQL
	templateConfig := map[string]interface{}{
		"mysql_version":           "8.0",
		"charset":                 "utf8mb4",
		"collation":               "utf8mb4_unicode_ci",
		"max_connections":         "300",
		"max_allowed_packet":      "128M",
		"innodb_buffer_pool_size": "2G",
		"innodb_log_file_size":    "512M",
		"mysql_port":              "3306",
	}

	// Build environment variables
	buildVars := map[string]string{
		"TZ": "UTC",
	}

	// Create BuildContainerInfo without GitHub URL
	container := &dto.BuildContainerInfo{
		ProjectID:      12, // Test ID range: 1-100
		ContainerID:    2,
		Name:           "mysql-custom",
		Slug:           imageName,
		TemplateBody:   &templateContent,
		TemplateConfig: templateConfig,
		NeedsBuild:     true,
		BuildVars:      buildVars,
	}

	// Create mock repository
	mockRepo := newMockBuildHistoryRepository()

	// Create BuildHistory record
	bh := build_history.NewBuildHistory(container.ContainerID)
	err = mockRepo.Create(ctx, bh)
	require.NoError(t, err, "Failed to create BuildHistory")

	buildHistoryID := bh.BuildHistoryID
	t.Logf("Created BuildHistory with ID: %d", buildHistoryID)

	// Create real clients
	tektonClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create TektonBuildClient: %v", err)
	}

	kubeClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create KubeBuildClient: %v", err)
	}

	// Create BuildService
	buildService := service.NewBuildService(
		mockRepo,
		tektonClient,
		kubeClient,
		logger.NewForTest(),
	)

	// Execute build
	t.Logf("Starting build for container: %s", container.Name)
	startTime := time.Now()

	result, err := buildService.BuildContainer(ctx, buildHistoryID, container)

	elapsed := time.Since(startTime)
	t.Logf("Build completed in %v", elapsed)

	// Verify result
	require.NoError(t, err, "BuildContainer should not return error")
	require.NotNil(t, result, "BuildResult should not be nil")

	assert.Equal(t, buildHistoryID, result.BuildHistoryID, "BuildHistoryID should match")
	assert.Equal(t, "success", result.Status, "Build should succeed")
	assert.True(t, result.ShouldBuild, "ShouldBuild should be true")
	assert.Equal(t, "latest", result.ImageTag, "ImageTag should be 'latest' for no-GitHub builds")
	assert.Empty(t, result.ErrorMessage, "ErrorMessage should be empty")
	assert.Empty(t, result.LatestCommitHash, "LatestCommitHash should be empty for no-GitHub builds")

	t.Logf("Build Result: Status=%s, CommitHash=%s, ImageTag=%s",
		result.Status, result.LatestCommitHash, result.ImageTag)

	// Verify BuildHistory was updated
	updatedBH, err := mockRepo.FindByID(ctx, buildHistoryID)
	require.NoError(t, err, "Should find updated BuildHistory")
	assert.Equal(t, build_history.BuildHistoryStatusSuccess, updatedBH.Status(), "BuildHistory status should be success")
}

// TestBuildServiceIntegration_SpringMySQLDemo tests BuildService with a private Spring Boot Gradle project.
// This test requires actual Tekton and Kubernetes infrastructure to be available.
//
// Prerequisites:
// - Tekton EventListener for builds must be accessible
// - Kubernetes API server must be accessible
// - Environment variables must be set in .env.test
// - GITHUB_APP_INSTALLATION_ID must be set for private repository access
func TestBuildServiceIntegration_SpringMySQLDemo(t *testing.T) {
	t.Parallel() // Enable parallel execution for long-running build test

	ctx := setupBuildServiceIntegrationTest(t)

	// Check GITHUB_APP_INSTALLATION_ID is set (required for private repository)
	installationIDStr := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	if installationIDStr == "" {
		t.Skip("Skipping test: GITHUB_APP_INSTALLATION_ID not set in environment")
	}

	var installationID int64
	_, err := fmt.Sscanf(installationIDStr, "%d", &installationID)
	if err != nil {
		t.Skipf("Skipping test: invalid GITHUB_APP_INSTALLATION_ID format: %v", err)
	}

	// Test configuration
	githubURL := "https://github.com/swm-launchpad/spring-mysql-demo.git"
	githubBranch := "main"
	imageName := "spring-mysql-demo-buildservice-test-01"
	directoryPath := "."
	templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/springboot-gradle.dockerfile.tmpl"

	// Read template
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Skipf("Skipping test: template file not found: %v", err)
	}
	templateContent := string(templateBytes)

	// Template configuration
	templateConfig := map[string]interface{}{
		"gradle_version": "8.5",
		"java_version":   "21",
		"app_port":       "8080",
	}

	// Build environment variables
	buildVars := map[string]string{
		"TZ":   "Asia/Seoul",
		"LANG": "en_US.UTF-8",
	}

	// Create BuildContainerInfo for private repository
	container := &dto.BuildContainerInfo{
		ProjectID:        13, // Test ID range: 1-100
		ContainerID:      3,
		Name:             "spring-mysql-demo",
		Slug:             imageName,
		TemplateBody:     &templateContent,
		TemplateConfig:   templateConfig,
		GitRepositoryURL: githubURL,
		GitBranch:        githubBranch,
		GitDirectoryPath: &directoryPath,
		NeedsBuild:       true,
		BuildVars:        buildVars,
		InstallationID:   &installationID,
	}

	// Create mock repository
	mockRepo := newMockBuildHistoryRepository()

	// Create BuildHistory record
	bh := build_history.NewBuildHistory(container.ContainerID)
	err = mockRepo.Create(ctx, bh)
	require.NoError(t, err, "Failed to create BuildHistory")

	buildHistoryID := bh.BuildHistoryID
	t.Logf("Created BuildHistory with ID: %d", buildHistoryID)

	// Create real clients
	tektonClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create TektonBuildClient: %v", err)
	}

	kubeClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
	if err != nil {
		t.Skipf("Skipping test: Failed to create KubeBuildClient: %v", err)
	}

	// Create BuildService
	buildService := service.NewBuildService(
		mockRepo,
		tektonClient,
		kubeClient,
		logger.NewForTest(),
	)

	// Execute build
	t.Logf("Starting build for container: %s (private repository)", container.Name)
	startTime := time.Now()

	result, err := buildService.BuildContainer(ctx, buildHistoryID, container)

	elapsed := time.Since(startTime)
	t.Logf("Build completed in %v", elapsed)

	// Verify result
	require.NoError(t, err, "BuildContainer should not return error")
	require.NotNil(t, result, "BuildResult should not be nil")

	assert.Equal(t, buildHistoryID, result.BuildHistoryID, "BuildHistoryID should match")
	assert.Equal(t, "success", result.Status, "Build should succeed")
	assert.True(t, result.ShouldBuild, "ShouldBuild should be true")
	assert.NotEmpty(t, result.LatestCommitHash, "LatestCommitHash should be set")
	assert.NotEmpty(t, result.ImageTag, "ImageTag should be set")
	assert.Empty(t, result.ErrorMessage, "ErrorMessage should be empty")

	t.Logf("Build Result: Status=%s, CommitHash=%s, ImageTag=%s",
		result.Status, result.LatestCommitHash, result.ImageTag)

	// Verify BuildHistory was updated
	updatedBH, err := mockRepo.FindByID(ctx, buildHistoryID)
	require.NoError(t, err, "Should find updated BuildHistory")
	assert.Equal(t, build_history.BuildHistoryStatusSuccess, updatedBH.Status(), "BuildHistory status should be success")
}
