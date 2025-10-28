package build

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	templatemodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template"
	templatevalue "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestGetContainersForBuildUseCase_Execute_Success(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	templateID := uint(1)
	templateBody := "FROM golang:1.21\nCOPY . /app"

	// Create mock containers
	mockContainers := []*containermodel.Container{
		createBuildContainerWithTemplate(1, projectID, &templateID),
		createBuildContainerWithoutTemplate(2, projectID),
	}

	// Create mock template
	mockTemplate := createMockTemplate(templateID, &templateBody)

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)
	mockTemplateRepo.On("FindByID", ctx, templateID).Return(mockTemplate, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 2)

	// Verify first container (with template)
	c1 := output.Containers[0]
	assert.Equal(t, uint(1), c1.ContainerID)
	assert.Equal(t, "build-container-with-template", c1.Name)
	assert.Equal(t, "c2025011812000011111111", c1.Slug)
	assert.NotNil(t, c1.TemplateID)
	assert.Equal(t, templateID, *c1.TemplateID)
	assert.NotNil(t, c1.TemplateBody)
	assert.Equal(t, templateBody, *c1.TemplateBody)
	assert.Equal(t, "https://github.com/test/repo", c1.GitRepositoryURL)
	assert.Equal(t, "main", c1.GitBranch)
	assert.NotNil(t, c1.LastBuiltCommitHash)
	assert.Equal(t, "abc1234567890", *c1.LastBuiltCommitHash)
	assert.True(t, c1.NeedsBuild)
	assert.Len(t, c1.BuildVars, 2)
	assert.Equal(t, "production", c1.BuildVars["BUILD_ENV"])
	assert.Equal(t, "1.21", c1.BuildVars["GO_VERSION"])

	// Verify second container (without template)
	c2 := output.Containers[1]
	assert.Equal(t, uint(2), c2.ContainerID)
	assert.Equal(t, "build-container-without-template", c2.Name)
	assert.Nil(t, c2.TemplateID)
	assert.Nil(t, c2.TemplateBody)
	assert.Len(t, c2.BuildVars, 0)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_EmptyList(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	mockContainers := []*containermodel.Container{}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 0)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_ServiceError(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(nil, assert.AnError)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_TemplateNotFound(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	templateID := uint(1)
	mockContainers := []*containermodel.Container{
		createBuildContainerWithTemplate(1, projectID, &templateID),
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)
	mockTemplateRepo.On("FindByID", ctx, templateID).Return(nil, containererrors.ErrTemplateNotFound)

	// Should return error when template is not found (strict error handling)
	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.ErrorIs(t, err, containererrors.ErrTemplateNotFound)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_TemplateRepositoryError(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	templateID := uint(1)
	mockContainers := []*containermodel.Container{
		createBuildContainerWithTemplate(1, projectID, &templateID),
	}

	// Simulate infrastructure error (DB connection failure, etc.)
	infraError := assert.AnError

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)
	mockTemplateRepo.On("FindByID", ctx, templateID).Return(nil, infraError)

	// Should return error when infrastructure fails (not silently ignore)
	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.ErrorIs(t, err, infraError)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_WithGitDirectoryPath(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	dirPath := "backend"
	mockContainers := []*containermodel.Container{
		createBuildContainerWithGitDirectory(1, projectID, &dirPath),
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 1)
	assert.NotNil(t, output.Containers[0].GitDirectoryPath)
	assert.Equal(t, dirPath, *output.Containers[0].GitDirectoryPath)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

func TestGetContainersForBuildUseCase_Execute_WithGitHubInstallationID(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	mockTemplateRepo := new(infrastructure.MockTemplateRepository)
	useCase := NewGetContainersForBuildUseCase(mockService, mockTemplateRepo)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForBuildInput{
		ProjectID: projectID,
	}

	installationID := int64(12345678)
	mockContainers := []*containermodel.Container{
		createBuildContainerWithInstallation(1, projectID, &installationID),
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 1)
	assert.NotNil(t, output.Containers[0].InstallationID)
	assert.Equal(t, installationID, *output.Containers[0].InstallationID)

	mockService.AssertExpectations(t)
	mockTemplateRepo.AssertExpectations(t)
}

// Helper functions for creating test containers

func createBuildContainerWithTemplate(containerID, projectID uint, templateID *uint) *containermodel.Container {
	slug, _ := value.NewContainerSlug("c2025011812000011111111")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)
	commitHash := "abc1234567890"
	templateConfig := map[string]interface{}{
		"go_version": "1.21",
	}

	c := containermodel.ReconstructContainer(
		containerID,
		projectID,
		templateID,
		"build-container-with-template",
		slug,
		nil,
		templateConfig,
		nil, // githubInstallationID
		gitConfig,
		nil,
		&commitHash,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	// Add build vars
	buildVar1 := containermodel.ReconstructBuildVar(1, containerID, "BUILD_ENV", "production", time.Now(), time.Now())
	buildVar2 := containermodel.ReconstructBuildVar(2, containerID, "GO_VERSION", "1.21", time.Now(), time.Now())
	_ = c.AddBuildVarDirect(buildVar1)
	_ = c.AddBuildVarDirect(buildVar2)

	return c
}

func createBuildContainerWithoutTemplate(containerID, projectID uint) *containermodel.Container {
	slug, _ := value.NewContainerSlug("c2025011812000022222222")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "develop", nil)
	cpuLimit := uint32(500)
	memoryLimit := uint32(1024)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := containermodel.ReconstructContainer(
		containerID,
		projectID,
		nil, // No template
		"build-container-without-template",
		slug,
		nil,
		nil,
		nil, // githubInstallationID
		gitConfig,
		nil,
		nil,
		false, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

func createBuildContainerWithGitDirectory(containerID, projectID uint, dirPath *string) *containermodel.Container {
	slug, _ := value.NewContainerSlug("c2025011812000033333333")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/monorepo", "main", dirPath)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := containermodel.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"build-container-with-directory",
		slug,
		nil,
		nil,
		nil, // githubInstallationID
		gitConfig,
		nil,
		nil,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

func createBuildContainerWithInstallation(containerID, projectID uint, installationID *int64) *containermodel.Container {
	slug, _ := value.NewContainerSlug("c2025011812000044444444")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/private-repo", "main", nil)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := containermodel.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"build-container-with-installation",
		slug,
		nil,
		nil,
		installationID,
		gitConfig,
		nil,
		nil,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

func createMockTemplate(templateID uint, templateBody *string) *templatemodel.Template {
	status, _ := templatevalue.NewTemplateStatus("active")
	config := &templatevalue.TemplateConfig{
		Description: "Go application template",
		Categories:  []string{"language", "backend"},
	}

	return templatemodel.ReconstructTemplate(
		templateID,
		"Go Template",
		templateBody,
		config,
		status,
		time.Now(),
		time.Now(),
	)
}

func TestDeepCopyTemplateConfig_NilInput(t *testing.T) {
	// nil input should return nil to preserve nil/empty distinction for Tekton
	result := deepCopyTemplateConfig(nil)
	assert.Nil(t, result)
}

func TestDeepCopyTemplateConfig_EmptyMap(t *testing.T) {
	// Empty map should return empty map (not nil)
	src := make(map[string]interface{})
	result := deepCopyTemplateConfig(src)

	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestDeepCopyTemplateConfig_SimpleMap(t *testing.T) {
	// Simple map should be deep copied
	src := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": true,
	}

	result := deepCopyTemplateConfig(src)

	assert.NotNil(t, result)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	assert.Equal(t, true, result["key3"])

	// Verify it's a deep copy by modifying the result
	result["key1"] = "modified"
	assert.Equal(t, "value1", src["key1"], "Original should not be affected")
}

func TestDeepCopyTemplateConfig_NestedMap(t *testing.T) {
	// Nested structures should be deep copied
	src := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": "5432",
		},
		"features": []interface{}{"auth", "logging"},
	}

	result := deepCopyTemplateConfig(src)

	assert.NotNil(t, result)

	// Verify values are copied correctly
	dbMap := result["database"].(map[string]interface{})
	assert.Equal(t, "localhost", dbMap["host"])
	assert.Equal(t, "5432", dbMap["port"])

	features := result["features"].([]interface{})
	assert.Equal(t, "auth", features[0])
	assert.Equal(t, "logging", features[1])

	// Verify nested map is deep copied
	dbMap["host"] = "modified"

	originalNested := src["database"].(map[string]interface{})
	assert.Equal(t, "localhost", originalNested["host"], "Original nested map should not be affected")

	// Verify nested slice is deep copied
	features[0] = "modified"

	originalSlice := src["features"].([]interface{})
	assert.Equal(t, "auth", originalSlice[0], "Original nested slice should not be affected")
}
