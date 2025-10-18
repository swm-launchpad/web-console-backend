package infrastructure

import (
	"context"
	"fmt"

	containerdeployment "github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// containerClient is the implementation of ContainerClient interface.
// It fetches actual container configuration from the container bounded context.
type containerClient struct {
	getContainersUseCase *containerdeployment.GetContainersForDeploymentUseCase
}

// NewContainerClient creates a new containerClient instance.
func NewContainerClient(
	getContainersUseCase *containerdeployment.GetContainersForDeploymentUseCase,
) projectinfra.ContainerClient {
	return &containerClient{
		getContainersUseCase: getContainersUseCase,
	}
}

// GetContainerConfig returns container configuration from the container bounded context.
func (c *containerClient) GetContainerConfig(ctx context.Context, projectID uint) (*dto.ContainerDeploymentConfig, error) {
	// Step 1: Get containers from container bounded context
	containersOutput, err := c.getContainersUseCase.Execute(ctx, containerdeployment.GetContainersForDeploymentInput{
		ProjectID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %w", err)
	}

	if len(containersOutput.Containers) == 0 {
		return nil, projecterrors.ErrContainerConfigNotFound
	}

	// Step 2: Convert to DTO format
	containerInfos := make([]dto.ContainerInfo, 0, len(containersOutput.Containers))

	for _, container := range containersOutput.Containers {
		// Image name: use container slug
		imageName := container.Slug

		// Image tag: use first 7 characters of last_built_git_commit_hash
		var imageTag string
		if container.LastBuiltGitCommitHash != nil && *container.LastBuiltGitCommitHash != "" {
			commitHash := *container.LastBuiltGitCommitHash
			if len(commitHash) >= 7 {
				imageTag = commitHash[:7]
			} else {
				imageTag = commitHash
			}
		} else {
			// No last built commit hash - cannot deploy
			return nil, fmt.Errorf("container %s has no last_built_git_commit_hash", container.Name)
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

	return &dto.ContainerDeploymentConfig{
		Containers: containerInfos,
	}, nil
}

// Compile-time assertion that containerClient implements ContainerClient interface
var _ projectinfra.ContainerClient = (*containerClient)(nil)
