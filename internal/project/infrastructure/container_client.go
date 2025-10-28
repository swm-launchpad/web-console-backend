package infrastructure

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containerbuild "github.com/swm-launchpad/web-console-backend/internal/container/application/build"
	containercombined "github.com/swm-launchpad/web-console-backend/internal/container/application/combined"
	containerdeployment "github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"go.uber.org/zap"
)

// containerClient is the implementation of ContainerClient interface.
// It fetches actual container configuration from the container bounded context.
type containerClient struct {
	getContainersForDeploymentUseCase     *containerdeployment.GetContainersForDeploymentUseCase
	getContainersForBuildUseCase          *containerbuild.GetContainersForBuildUseCase
	getContainersForBuildAndDeployUseCase *containercombined.GetContainersForBuildAndDeployUseCase
	logger                                logger.Logger
}

// NewContainerClient creates a new containerClient instance.
func NewContainerClient(
	getContainersForDeploymentUseCase *containerdeployment.GetContainersForDeploymentUseCase,
	getContainersForBuildUseCase *containerbuild.GetContainersForBuildUseCase,
	getContainersForBuildAndDeployUseCase *containercombined.GetContainersForBuildAndDeployUseCase,
	log logger.Logger,
) projectinfra.ContainerClient {
	return &containerClient{
		getContainersForDeploymentUseCase:     getContainersForDeploymentUseCase,
		getContainersForBuildUseCase:          getContainersForBuildUseCase,
		getContainersForBuildAndDeployUseCase: getContainersForBuildAndDeployUseCase,
		logger:                                log,
	}
}

// GetContainerConfig returns container configuration from the container bounded context.
func (c *containerClient) GetContainerConfig(ctx context.Context, projectID uint) (*dto.ContainerDeploymentConfig, error) {
	c.logger.Info(ctx, "container client get container config started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Get containers from container bounded context
	containersOutput, err := c.getContainersForDeploymentUseCase.Execute(ctx, containerdeployment.GetContainersForDeploymentInput{
		ProjectID: projectID,
	})
	if err != nil {
		c.logger.Error(ctx, "container client failed to get containers",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	if len(containersOutput.Containers) == 0 {
		c.logger.Warn(ctx, "container client no containers found",
			zap.Uint("project_id", projectID),
		)
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 2: Convert to DTO format
	containerInfos := make([]dto.ContainerInfo, 0, len(containersOutput.Containers))

	for _, container := range containersOutput.Containers {
		// Image name: use container slug
		imageName := container.Slug

		// Image tag: use first 7 characters of last_built_git_commit_hash
		// For first-time builds, last_built_git_commit_hash is NULL, so we use a placeholder
		// that will be updated from BuildResult before deployment
		var imageTag string
		if container.LastBuiltGitCommitHash != nil && *container.LastBuiltGitCommitHash != "" {
			commitHash := *container.LastBuiltGitCommitHash
			if len(commitHash) >= 7 {
				imageTag = commitHash[:7]
			} else {
				imageTag = commitHash
			}
		} else {
			// First-time build: use placeholder tag
			// This will be updated from BuildResult.LatestCommitHash before deployment
			imageTag = "pending"
			c.logger.Info(ctx, "container client using placeholder image tag for first build",
				zap.Uint("project_id", projectID),
				zap.String("container_name", container.Name),
				zap.String("placeholder_tag", imageTag),
			)
		}

		// Health check: always "none"
		healthCheckType := "none"

		// Port and Domain: from first network (assuming only one network per container)
		var port int
		var domain *string
		if len(container.Networks) > 0 {
			network := container.Networks[0]
			port = int(network.InternalPort)

			if network.FQDN != "" {
				domain = &network.FQDN
			}
		} else {
			// No network defined - cannot deploy
			c.logger.Error(ctx, "container client container missing network configuration",
				zap.Uint("project_id", projectID),
				zap.String("container_name", container.Name),
			)
			return nil, fmt.Errorf("container %s has no network configuration", container.Name)
		}

		// Resource limits
		cpuLimit := "1000m" // default
		if container.CPULimit != nil {
			cpuLimit = fmt.Sprintf("%dm", *container.CPULimit)
		}

		memoryLimit := "1Gi" // default
		if container.MemoryLimit != nil {
			memoryLimit = fmt.Sprintf("%dMi", *container.MemoryLimit)
		}
		// MemoryRequest = MemoryLimit (same value)
		memoryRequest := memoryLimit

		// Volume mounts: pass volume_id as-is
		// The actual volume_id to volume_slug mapping will be done at the deployment service layer
		volumeMounts := make([]dto.VolumeMount, 0, len(container.Mounts))
		for _, mount := range container.Mounts {
			volumeMounts = append(volumeMounts, dto.VolumeMount{
				VolumeID:  mount.VolumeID,
				MountPath: mount.MountPath,
			})
		}

		// Build ContainerInfo
		containerInfo := dto.ContainerInfo{
			Name:            container.Slug, // Use slug as container name for Kubernetes
			Domain:          domain,
			HealthCheckType: healthCheckType,
			HealthEndpoint:  nil, // No health endpoint for "none" type
			Port:            port,
			HealthPort:      nil, // No separate health port
			ImageName:       imageName,
			ImageTag:        imageTag,
			EnvVars:         container.EnvVars,
			Secrets:         container.Secrets,
			CPULimit:        cpuLimit,
			MemoryRequest:   memoryRequest,
			MemoryLimit:     memoryLimit,
			VolumeMounts:    volumeMounts,
		}

		containerInfos = append(containerInfos, containerInfo)
	}

	c.logger.Info(ctx, "container client get container config completed",
		zap.Uint("project_id", projectID),
		zap.Int("container_count", len(containerInfos)),
	)

	return &dto.ContainerDeploymentConfig{
		Containers: containerInfos,
	}, nil
}

// GetContainerBuildConfig returns container build configuration from the container bounded context.
func (c *containerClient) GetContainerBuildConfig(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, error) {
	c.logger.Info(ctx, "container client get container build config started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Get containers from container bounded context
	containersOutput, err := c.getContainersForBuildUseCase.Execute(ctx, containerbuild.GetContainersForBuildInput{
		ProjectID: projectID,
	})
	if err != nil {
		c.logger.Error(ctx, "container client failed to get containers for build",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get containers for build: %w", err)
	}

	if len(containersOutput.Containers) == 0 {
		c.logger.Warn(ctx, "container client no containers found for build",
			zap.Uint("project_id", projectID),
		)
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 2: Convert to DTO format
	containerInfos := make([]dto.BuildContainerInfo, 0, len(containersOutput.Containers))

	for _, container := range containersOutput.Containers {
		containerInfo := dto.BuildContainerInfo{
			ProjectID:           uint(projectID),
			ContainerID:         container.ContainerID,
			Name:                container.Name,
			Slug:                container.Slug,
			TemplateID:          container.TemplateID,
			TemplateBody:        container.TemplateBody,
			TemplateConfig:      container.TemplateConfig,
			GitRepositoryURL:    container.GitRepositoryURL,
			GitBranch:           container.GitBranch,
			GitDirectoryPath:    container.GitDirectoryPath,
			LastBuiltCommitHash: container.LastBuiltCommitHash,
			NeedsBuild:          container.NeedsBuild,
			BuildVars:           container.BuildVars,
			InstallationID:      container.InstallationID,
		}

		containerInfos = append(containerInfos, containerInfo)
	}

	c.logger.Info(ctx, "container client get container build config completed",
		zap.Uint("project_id", projectID),
		zap.Int("container_count", len(containerInfos)),
	)

	return &dto.ContainerBuildConfig{
		Containers: containerInfos,
	}, nil
}

// GetContainerConfigs returns both build and deployment container configurations
// using a single database query. This ensures perfect snapshot consistency and
// eliminates the risk of configuration divergence between build and deployment.
func (c *containerClient) GetContainerConfigs(ctx context.Context, projectID uint) (
	*dto.ContainerBuildConfig,
	*dto.ContainerDeploymentConfig,
	error,
) {
	c.logger.Info(ctx, "container client get container configs (unified) started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Get containers from container bounded context using unified use case
	// This executes a single database query and transforms to both formats
	combined, err := c.getContainersForBuildAndDeployUseCase.Execute(ctx, containercombined.GetContainersForBuildAndDeployInput{
		ProjectID: projectID,
	})
	if err != nil {
		c.logger.Error(ctx, "container client failed to get containers (unified)",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, nil, fmt.Errorf("failed to get containers: %w", err)
	}

	if len(combined.BuildContainers) == 0 {
		c.logger.Warn(ctx, "container client no containers found (unified)",
			zap.Uint("project_id", projectID),
		)
		return nil, nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 2: Convert build containers to project DTO format
	buildContainerInfos := make([]dto.BuildContainerInfo, 0, len(combined.BuildContainers))

	for _, container := range combined.BuildContainers {
		containerInfo := dto.BuildContainerInfo{
			ProjectID:           projectID,
			ContainerID:         container.ContainerID,
			Name:                container.Name,
			Slug:                container.Slug,
			TemplateID:          container.TemplateID,
			TemplateBody:        container.TemplateBody,
			TemplateConfig:      container.TemplateConfig,
			GitRepositoryURL:    container.GitRepositoryURL,
			GitBranch:           container.GitBranch,
			GitDirectoryPath:    container.GitDirectoryPath,
			LastBuiltCommitHash: container.LastBuiltCommitHash,
			NeedsBuild:          container.NeedsBuild,
			BuildVars:           container.BuildVars,
			InstallationID:      container.InstallationID,
		}

		buildContainerInfos = append(buildContainerInfos, containerInfo)
	}

	// Step 3: Convert deployment containers to project DTO format
	deployContainerInfos := make([]dto.ContainerInfo, 0, len(combined.DeploymentContainers))

	for _, container := range combined.DeploymentContainers {
		// Image name: use container slug
		imageName := container.Slug

		// Image tag: use first 7 characters of last_built_git_commit_hash
		// For first-time builds, last_built_git_commit_hash is NULL, so we use a placeholder
		// that will be updated from BuildResult before deployment
		var imageTag string
		if container.LastBuiltGitCommitHash != nil && *container.LastBuiltGitCommitHash != "" {
			commitHash := *container.LastBuiltGitCommitHash
			if len(commitHash) >= 7 {
				imageTag = commitHash[:7]
			} else {
				imageTag = commitHash
			}
		} else {
			// First-time build: use placeholder tag
			// This will be updated from BuildResult.LatestCommitHash before deployment
			imageTag = "pending"
			c.logger.Info(ctx, "container client using placeholder image tag for first build (unified)",
				zap.Uint("project_id", projectID),
				zap.String("container_name", container.Name),
				zap.String("placeholder_tag", imageTag),
			)
		}

		// Health check: always "none"
		healthCheckType := "none"

		// Port and Domain: from first network (assuming only one network per container)
		var port int
		var domain *string
		if len(container.Networks) > 0 {
			network := container.Networks[0]
			port = int(network.InternalPort)

			if network.FQDN != "" {
				domain = &network.FQDN
			}
		} else {
			// No network defined - cannot deploy
			c.logger.Error(ctx, "container client container missing network configuration (unified)",
				zap.Uint("project_id", projectID),
				zap.String("container_name", container.Name),
			)
			return nil, nil, fmt.Errorf("container %s has no network configuration", container.Name)
		}

		// Resource limits
		cpuLimit := "1000m" // default
		if container.CPULimit != nil {
			cpuLimit = fmt.Sprintf("%dm", *container.CPULimit)
		}

		memoryLimit := "1Gi" // default
		if container.MemoryLimit != nil {
			memoryLimit = fmt.Sprintf("%dMi", *container.MemoryLimit)
		}
		// MemoryRequest = MemoryLimit (same value)
		memoryRequest := memoryLimit

		// Volume mounts: pass volume_id as-is
		// The actual volume_id to volume_slug mapping will be done at the deployment service layer
		volumeMounts := make([]dto.VolumeMount, 0, len(container.Mounts))
		for _, mount := range container.Mounts {
			volumeMounts = append(volumeMounts, dto.VolumeMount{
				VolumeID:  mount.VolumeID,
				MountPath: mount.MountPath,
			})
		}

		// Build ContainerInfo
		containerInfo := dto.ContainerInfo{
			Name:            container.Slug, // Use slug as container name for Kubernetes
			Domain:          domain,
			HealthCheckType: healthCheckType,
			HealthEndpoint:  nil, // No health endpoint for "none" type
			Port:            port,
			HealthPort:      nil, // No separate health port
			ImageName:       imageName,
			ImageTag:        imageTag,
			EnvVars:         container.EnvVars,
			Secrets:         container.Secrets,
			CPULimit:        cpuLimit,
			MemoryRequest:   memoryRequest,
			MemoryLimit:     memoryLimit,
			VolumeMounts:    volumeMounts,
		}

		deployContainerInfos = append(deployContainerInfos, containerInfo)
	}

	c.logger.Info(ctx, "container client get container configs (unified) completed",
		zap.Uint("project_id", projectID),
		zap.Int("build_container_count", len(buildContainerInfos)),
		zap.Int("deploy_container_count", len(deployContainerInfos)),
	)

	return &dto.ContainerBuildConfig{
			Containers: buildContainerInfos,
		}, &dto.ContainerDeploymentConfig{
			Containers: deployContainerInfos,
		}, nil
}

// Compile-time assertion that containerClient implements ContainerClient interface
var _ projectinfra.ContainerClient = (*containerClient)(nil)
