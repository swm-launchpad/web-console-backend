package deployment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestGetContainersForDeploymentUseCase_Execute_Success(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	useCase := NewGetContainersForDeploymentUseCase(mockService)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForDeploymentInput{
		ProjectID: projectID,
	}

	// Create mock containers with various configurations
	mockContainers := []*model.Container{
		createContainerWithAllFields(1, projectID),
		createContainerWithMinimalFields(2, projectID),
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 2)

	// Verify first container (with all fields)
	c1 := output.Containers[0]
	assert.Equal(t, uint(1), c1.ContainerID)
	assert.Equal(t, "container-with-all", c1.Name)
	assert.Equal(t, "c2025011812000011111111", c1.Slug)
	assert.NotNil(t, c1.LastBuiltGitCommitHash)
	assert.Equal(t, "abc1234567890", *c1.LastBuiltGitCommitHash)
	assert.NotNil(t, c1.CPULimit)
	assert.Equal(t, uint32(1000), *c1.CPULimit)
	assert.NotNil(t, c1.MemoryLimit)
	assert.Equal(t, uint32(2048), *c1.MemoryLimit)
	assert.Len(t, c1.EnvVars, 2)
	assert.Equal(t, "production", c1.EnvVars["ENV"])
	assert.Equal(t, "postgres://db", c1.EnvVars["DATABASE_URL"])
	assert.Len(t, c1.Secrets, 1)
	assert.Equal(t, "secret123", c1.Secrets["API_KEY"])
	assert.Len(t, c1.Networks, 1)
	assert.Equal(t, uint16(8080), c1.Networks[0].InternalPort)
	assert.Equal(t, "tcp", c1.Networks[0].NetworkType)
	assert.Len(t, c1.Mounts, 1)
	assert.Equal(t, uint(1), c1.Mounts[0].VolumeID)
	assert.Equal(t, "/data", c1.Mounts[0].MountPath)

	// Verify second container (minimal fields)
	c2 := output.Containers[1]
	assert.Equal(t, uint(2), c2.ContainerID)
	assert.Equal(t, "minimal-container", c2.Name)
	assert.Nil(t, c2.LastBuiltGitCommitHash)
	assert.Len(t, c2.EnvVars, 0)
	assert.Len(t, c2.Secrets, 0)
	assert.Len(t, c2.Networks, 0)
	assert.Len(t, c2.Mounts, 0)

	mockService.AssertExpectations(t)
}

func TestGetContainersForDeploymentUseCase_Execute_EmptyList(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	useCase := NewGetContainersForDeploymentUseCase(mockService)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForDeploymentInput{
		ProjectID: projectID,
	}

	mockContainers := []*model.Container{}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 0)

	mockService.AssertExpectations(t)
}

func TestGetContainersForDeploymentUseCase_Execute_ServiceError(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	useCase := NewGetContainersForDeploymentUseCase(mockService)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForDeploymentInput{
		ProjectID: projectID,
	}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(nil, assert.AnError)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockService.AssertExpectations(t)
}

func TestGetContainersForDeploymentUseCase_Execute_WithNetworkOptionalFields(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	useCase := NewGetContainersForDeploymentUseCase(mockService)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForDeploymentInput{
		ProjectID: projectID,
	}

	// Create container with network having optional fields
	container := createContainerWithNetworkOptionalFields(1, projectID)
	mockContainers := []*model.Container{container}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 1)

	// Verify network optional fields
	c := output.Containers[0]
	assert.Len(t, c.Networks, 1)
	network := c.Networks[0]
	assert.Equal(t, uint16(8080), network.InternalPort)
	assert.Equal(t, uint16(18080), network.ExternalPort)
	assert.Equal(t, "192.168.1.100", network.ExternalIP)
	assert.Equal(t, "app.example.com", network.FQDN)
	assert.Equal(t, "tcp", network.NetworkType)

	mockService.AssertExpectations(t)
}

func TestGetContainersForDeploymentUseCase_Execute_WithMultipleMounts(t *testing.T) {
	mockService := new(infrastructure.MockContainerService)
	useCase := NewGetContainersForDeploymentUseCase(mockService)

	ctx := context.Background()
	projectID := uint(10)

	input := GetContainersForDeploymentInput{
		ProjectID: projectID,
	}

	// Create container with multiple mounts
	container := createContainerWithMultipleMounts(1, projectID)
	mockContainers := []*model.Container{container}

	mockService.On("ListContainersByProjectID", ctx, projectID).Return(mockContainers, nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 1)

	// Verify multiple mounts
	c := output.Containers[0]
	assert.Len(t, c.Mounts, 3)
	assert.Equal(t, uint(1), c.Mounts[0].VolumeID)
	assert.Equal(t, "/data", c.Mounts[0].MountPath)
	assert.Equal(t, uint(2), c.Mounts[1].VolumeID)
	assert.Equal(t, "/var/lib/mysql", c.Mounts[1].MountPath)
	assert.Equal(t, uint(3), c.Mounts[2].VolumeID)
	assert.Equal(t, "/var/log", c.Mounts[2].MountPath)

	mockService.AssertExpectations(t)
}

// Helper functions for creating test containers

func createContainerWithAllFields(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000011111111")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)
	commitHash := "abc1234567890"

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"container-with-all",
		slug,
		nil,
		nil,
		nil, // githubInstallationID
		gitConfig,
		nil,
		&commitHash,
		false, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		nil,   // webhookToken
		false, // webhookEnabled
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	// Add env vars
	envVar1 := model.ReconstructEnvVar(1, containerID, "ENV", "production", time.Now(), time.Now())
	envVar2 := model.ReconstructEnvVar(2, containerID, "DATABASE_URL", "postgres://db", time.Now(), time.Now())
	_ = c.AddEnvVarDirect(envVar1)
	_ = c.AddEnvVarDirect(envVar2)

	// Add secret
	secret1 := model.ReconstructSecret(1, containerID, "API_KEY", "secret123", time.Now(), time.Now())
	_ = c.AddSecretDirect(secret1)

	// Add network
	networkType, _ := value.NewNetworkType("tcp")
	internalPort := uint16(8080)
	network := model.ReconstructNetwork(1, containerID, &internalPort, nil, networkType, nil, nil, nil, nil, time.Now(), time.Now())
	_ = c.AddNetworkDirect(network)

	// Add mount
	mount := model.ReconstructMount(containerID, 1, "/data", time.Now(), time.Now())
	_ = c.AddMountDirect(mount)

	return c
}

func createContainerWithMinimalFields(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000022222222")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(500)
	memoryLimit := uint32(1024)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"minimal-container",
		slug,
		nil,
		nil,
		nil, // githubInstallationID
		gitConfig,
		nil, // No commit hash
		nil,
		false, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		nil,   // webhookToken
		false, // webhookEnabled
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

func createContainerWithNetworkOptionalFields(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000033333333")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"network-optional-container",
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
		nil,   // webhookToken
		false, // webhookEnabled
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	// Add network with all optional fields
	networkType, _ := value.NewNetworkType("tcp")
	internalPort := uint16(8080)
	externalPort := uint16(18080)
	externalIP := "192.168.1.100"
	fqdn := "app.example.com"
	network := model.ReconstructNetwork(
		1,
		containerID,
		&internalPort,
		&externalPort,
		networkType,
		&externalIP,
		&fqdn,
		nil,
		nil,
		time.Now(),
		time.Now(),
	)
	_ = c.AddNetworkDirect(network)

	return c
}

func createContainerWithMultipleMounts(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000044444444")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"multiple-mounts-container",
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
		nil,   // webhookToken
		false, // webhookEnabled
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	// Add multiple mounts
	mount1 := model.ReconstructMount(containerID, 1, "/data", time.Now(), time.Now())
	mount2 := model.ReconstructMount(containerID, 2, "/var/lib/mysql", time.Now(), time.Now())
	mount3 := model.ReconstructMount(containerID, 3, "/var/log", time.Now(), time.Now())
	_ = c.AddMountDirect(mount1)
	_ = c.AddMountDirect(mount2)
	_ = c.AddMountDirect(mount3)

	return c
}
