package combined

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/application/build"
	"github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// GetContainersForBuildAndDeployInput represents input for getting containers for both build and deployment
type GetContainersForBuildAndDeployInput struct {
	ProjectID uint
}

// GetContainersForBuildAndDeployOutput contains container information for both build and deployment
// This unified output ensures both snapshots are captured from the same database query,
// preventing divergence between build and deployment configurations
type GetContainersForBuildAndDeployOutput struct {
	BuildContainers      []build.BuildContainerOutput           `json:"build_containers"`
	DeploymentContainers []deployment.DeploymentContainerOutput `json:"deployment_containers"`
}

// GetContainersForBuildAndDeployUseCase executes a single database query to fetch containers
// and transforms them into both build and deployment formats.
// This ensures perfect snapshot consistency and eliminates the risk of configuration divergence.
type GetContainersForBuildAndDeployUseCase struct {
	containerService   service.ContainerService
	templateRepository repository.TemplateRepository
	logger             logger.Logger
}

// NewGetContainersForBuildAndDeployUseCase creates a new instance
func NewGetContainersForBuildAndDeployUseCase(
	containerService service.ContainerService,
	templateRepository repository.TemplateRepository,
	log logger.Logger,
) *GetContainersForBuildAndDeployUseCase {
	return &GetContainersForBuildAndDeployUseCase{
		containerService:   containerService,
		templateRepository: templateRepository,
		logger:             log,
	}
}

// Execute fetches containers once and transforms them into both build and deployment formats
func (uc *GetContainersForBuildAndDeployUseCase) Execute(
	ctx context.Context,
	input GetContainersForBuildAndDeployInput,
) (*GetContainersForBuildAndDeployOutput, error) {
	// Single database query - performance improvement and perfect consistency
	containers, err := uc.containerService.ListContainersByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Transform to both formats from the same data
	buildContainers := make([]build.BuildContainerOutput, 0, len(containers))
	deploymentContainers := make([]deployment.DeploymentContainerOutput, 0, len(containers))

	for _, container := range containers {
		// Build format transformation
		buildOutput, err := uc.transformToBuildOutput(ctx, container)
		if err != nil {
			return nil, err
		}
		buildContainers = append(buildContainers, buildOutput)

		// Deployment format transformation
		deploymentOutput := uc.transformToDeploymentOutput(container)
		deploymentContainers = append(deploymentContainers, deploymentOutput)
	}

	return &GetContainersForBuildAndDeployOutput{
		BuildContainers:      buildContainers,
		DeploymentContainers: deploymentContainers,
	}, nil
}

// transformToBuildOutput transforms a container to BuildContainerOutput
func (uc *GetContainersForBuildAndDeployUseCase) transformToBuildOutput(
	ctx context.Context,
	container *model.Container,
) (build.BuildContainerOutput, error) {
	// Build build vars map (defensive copy to prevent aliasing)
	buildVars := make(map[string]string)
	for _, bv := range container.BuildVars() {
		buildVars[bv.Key()] = bv.Value()
	}

	// Deep copy template config to prevent snapshot aliasing
	templateConfig, err := deepCopyTemplateConfig(container.TemplateConfig())
	if err != nil {
		// Log serialization error for diagnostic purposes
		uc.logger.Warn(ctx, "Failed to deep copy template config, using nil",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		templateConfig = nil
	}

	// Get template body if template is configured
	var templateBody *string
	if container.TemplateID() != nil {
		template, err := uc.templateRepository.FindByID(ctx, *container.TemplateID())
		if err != nil {
			// Template ID is set but template not found - this indicates data inconsistency
			return build.BuildContainerOutput{}, err
		}
		templateBody = template.TemplateBody()
	}

	return build.BuildContainerOutput{
		ContainerID:         container.ContainerID(),
		Name:                container.Name(),
		Slug:                container.Slug().String(),
		TemplateID:          container.TemplateID(),
		TemplateBody:        templateBody,
		TemplateConfig:      templateConfig,
		GitRepositoryURL:    container.GitConfig().RepositoryURL(),
		GitBranch:           container.GitConfig().Branch(),
		GitDirectoryPath:    container.GitConfig().DirectoryPath(),
		LastBuiltCommitHash: container.LastBuiltGitCommitHash(),
		NeedsBuild:          container.NeedsBuild(),
		BuildVars:           buildVars,
		InstallationID:      container.GitHubInstallationID(),
	}, nil
}

// transformToDeploymentOutput transforms a container to DeploymentContainerOutput
func (uc *GetContainersForBuildAndDeployUseCase) transformToDeploymentOutput(
	container *model.Container,
) deployment.DeploymentContainerOutput {
	// Build env vars map
	envVars := make(map[string]string)
	for _, ev := range container.EnvVars() {
		envVars[ev.Key()] = ev.Value()
	}

	// Build secrets map
	secrets := make(map[string]string)
	for _, s := range container.Secrets() {
		secrets[s.Key()] = s.Value()
	}

	// Build networks
	networks := make([]deployment.NetworkOutput, 0, len(container.Networks()))
	for _, network := range container.Networks() {
		netOutput := deployment.NetworkOutput{
			NetworkID:   network.NetworkID(),
			NetworkType: network.NetworkType().String(),
		}

		if internalPort := network.InternalPort(); internalPort != nil {
			netOutput.InternalPort = *internalPort
		}

		if externalPort := network.ExternalPort(); externalPort != nil {
			netOutput.ExternalPort = *externalPort
		}

		if externalIP := network.ExternalIP(); externalIP != nil {
			netOutput.ExternalIP = *externalIP
		}

		if fqdn := network.FQDN(); fqdn != nil {
			netOutput.FQDN = *fqdn
		}

		networks = append(networks, netOutput)
	}

	// Build mounts
	mounts := make([]deployment.MountOutput, 0, len(container.Mounts()))
	for _, mount := range container.Mounts() {
		mounts = append(mounts, deployment.MountOutput{
			VolumeID:  mount.VolumeID(),
			MountPath: mount.MountPath(),
		})
	}

	return deployment.DeploymentContainerOutput{
		ContainerID:            container.ContainerID(),
		Name:                   container.Name(),
		Slug:                   container.Slug().String(),
		GitHubInstallationID:   container.GitHubInstallationID(),
		LastBuiltGitCommitHash: container.LastBuiltGitCommitHash(),
		CPULimit:               container.ResourceLimits().CPULimit(),
		MemoryLimit:            container.ResourceLimits().MemoryLimit(),
		EnvVars:                envVars,
		Secrets:                secrets,
		Networks:               networks,
		Mounts:                 mounts,
	}
}

// deepCopyTemplateConfig performs a deep copy of template config using JSON serialization
func deepCopyTemplateConfig(src map[string]interface{}) (map[string]interface{}, error) {
	if src == nil {
		return nil, nil
	}

	// Use JSON round-trip for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	var dest map[string]interface{}
	if err := json.Unmarshal(data, &dest); err != nil {
		return nil, err
	}

	return dest, nil
}
