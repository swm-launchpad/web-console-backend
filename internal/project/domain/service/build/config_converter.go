package build

import (
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// ConvertToBuildConfig extracts build-specific fields from the unified container configuration.
// This conversion ensures that the build phase uses the exact same container snapshot
// that will be used for deployment later.
func ConvertToBuildConfig(unified *dto.UnifiedContainerConfig) *dto.ContainerBuildConfig {
	if unified == nil {
		return nil
	}

	buildConfig := &dto.ContainerBuildConfig{
		Containers: make([]dto.BuildContainerInfo, len(unified.Containers)),
	}

	for i, container := range unified.Containers {
		buildConfig.Containers[i] = dto.BuildContainerInfo{
			ProjectID:           unified.ProjectID,
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
			InstallationID:      container.GitHubInstallationID,
		}
	}

	return buildConfig
}

// ConvertToDeployConfig extracts deployment-specific fields from the unified container configuration.
// It optionally updates image tags based on build results, ensuring that newly built images are deployed.
//
// Parameters:
//   - unified: The unified container configuration (single source of truth)
//   - buildResults: Optional build results to update image tags. If nil, uses tags from unified config.
//
// Image tag update strategy:
//   - If buildResults is provided, map ContainerID → BuildResult
//   - For each container, if a successful build result exists with a commit hash, update the image tag
//   - Otherwise, use the image tag from the unified config (preserves snapshot consistency)
//
// This approach eliminates the divergence risk between build and deployment configurations
// by ensuring both phases use the same underlying container snapshot, with only image tags
// updated based on actual build outcomes.
func ConvertToDeployConfig(
	unified *dto.UnifiedContainerConfig,
	buildResults []*BuildResult,
) *dto.ContainerDeploymentConfig {
	if unified == nil {
		return nil
	}

	deployConfig := &dto.ContainerDeploymentConfig{
		Containers: make([]dto.ContainerInfo, len(unified.Containers)),
	}

	// Build map of ContainerID → BuildResult for efficient lookup
	buildResultByID := make(map[uint]*BuildResult)
	for _, result := range buildResults {
		if result != nil {
			buildResultByID[result.ContainerID] = result
		}
	}

	for i, container := range unified.Containers {
		// Start with base image tag from unified config
		imageTag := container.ImageTag

		// Update with build result if available
		// This ensures we deploy the newly built image while maintaining all other
		// configuration from the original snapshot
		if result, found := buildResultByID[container.ContainerID]; found {
			if result.Status == "success" && result.LatestCommitHash != "" {
				commitHash := result.LatestCommitHash
				if len(commitHash) >= 7 {
					imageTag = commitHash[:7]
				} else {
					imageTag = commitHash
				}
			}
		}
		// If build result not found, use imageTag from unified config
		// (either commit hash from previous build or "latest" for first-time builds)

		// Note: Networks are included in unified config but not in ContainerInfo DTO
		// Network information (domain, port) is extracted separately

		// Convert Mounts from unified format to deployment format
		mounts := make([]dto.VolumeMount, len(container.Mounts))
		for j, mount := range container.Mounts {
			mounts[j] = dto.VolumeMount(mount)
		}

		deployConfig.Containers[i] = dto.ContainerInfo{
			Name:            container.Slug, // Use slug as name for deployment
			Domain:          container.Domain,
			HealthCheckType: container.HealthCheckType,
			HealthEndpoint:  container.HealthEndpoint,
			Port:            container.Port,
			HealthPort:      container.HealthPort,
			ImageName:       container.ImageName,
			ImageTag:        imageTag, // Updated tag (either from build result or original)
			EnvVars:         container.EnvVars,
			Secrets:         container.Secrets,
			CPULimit:        formatCPULimit(container.CPULimit),
			MemoryRequest:   formatMemoryRequest(container.MemoryLimit),
			MemoryLimit:     formatMemoryLimit(container.MemoryLimit),
			VolumeMounts:    mounts,
		}
	}

	return deployConfig
}

// formatCPULimit converts CPU limit from millicores to Kubernetes format
func formatCPULimit(cpuLimit *uint32) string {
	if cpuLimit == nil {
		return "1000m" // Default 1 core
	}
	return formatMillicores(*cpuLimit)
}

// formatMemoryRequest converts memory limit to memory request (same value for now)
// Note: memoryLimit is already in MiB units, not bytes
func formatMemoryRequest(memoryLimit *uint32) string {
	if memoryLimit == nil {
		return "512Mi" // Default 512Mi
	}
	return fmt.Sprintf("%dMi", *memoryLimit)
}

// formatMemoryLimit converts memory limit to Kubernetes format
// Note: memoryLimit is already in MiB units, not bytes
func formatMemoryLimit(memoryLimit *uint32) string {
	if memoryLimit == nil {
		return "1Gi" // Default 1Gi
	}
	return fmt.Sprintf("%dMi", *memoryLimit)
}

// formatMillicores formats CPU millicores for Kubernetes
func formatMillicores(millicores uint32) string {
	return fmt.Sprintf("%dm", millicores)
}
