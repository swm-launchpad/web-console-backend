package deployment

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type GetContainersForDeploymentInput struct {
	ProjectID uint
}

// NetworkOutput represents network configuration for deployment
type NetworkOutput struct {
	NetworkID    uint   `json:"network_id"`
	InternalPort uint16 `json:"internal_port"`
	ExternalPort uint16 `json:"external_port,omitempty"`
	ExternalIP   string `json:"external_ip,omitempty"`
	FQDN         string `json:"fqdn,omitempty"`
	NetworkType  string `json:"network_type"`
}

// MountOutput represents mount configuration for deployment
type MountOutput struct {
	VolumeID  uint   `json:"volume_id"`
	MountPath string `json:"mount_path"`
}

type DeploymentContainerOutput struct {
	ContainerID            uint              `json:"container_id"`
	Name                   string            `json:"name"`
	Slug                   string            `json:"slug"`
	LastBuiltGitCommitHash *string           `json:"last_built_git_commit_hash,omitempty"`
	CPULimit               *uint32           `json:"cpu_limit,omitempty"`
	MemoryLimit            *uint32           `json:"memory_limit,omitempty"`
	EnvVars                map[string]string `json:"env_vars"`
	Secrets                map[string]string `json:"secrets"`
	Networks               []NetworkOutput   `json:"networks"`
	Mounts                 []MountOutput     `json:"mounts"`
}

type GetContainersForDeploymentOutput struct {
	Containers []DeploymentContainerOutput `json:"containers"`
}

type GetContainersForDeploymentUseCase struct {
	containerService service.ContainerService
}

func NewGetContainersForDeploymentUseCase(containerService service.ContainerService) *GetContainersForDeploymentUseCase {
	return &GetContainersForDeploymentUseCase{
		containerService: containerService,
	}
}

func (uc *GetContainersForDeploymentUseCase) Execute(ctx context.Context, input GetContainersForDeploymentInput) (*GetContainersForDeploymentOutput, error) {
	// Get all containers for the project via service
	containers, err := uc.containerService.ListContainersByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetContainersForDeploymentOutput{
		Containers: make([]DeploymentContainerOutput, 0, len(containers)),
	}

	for _, container := range containers {
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
		networks := make([]NetworkOutput, 0, len(container.Networks()))
		for _, network := range container.Networks() {
			netOutput := NetworkOutput{
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
		mounts := make([]MountOutput, 0, len(container.Mounts()))
		for _, mount := range container.Mounts() {
			mounts = append(mounts, MountOutput{
				VolumeID:  mount.VolumeID(),
				MountPath: mount.MountPath(),
			})
		}

		containerOutput := DeploymentContainerOutput{
			ContainerID:            container.ContainerID(),
			Name:                   container.Name(),
			Slug:                   container.Slug().String(),
			LastBuiltGitCommitHash: container.LastBuiltGitCommitHash(),
			CPULimit:               container.ResourceLimits().CPULimit(),
			MemoryLimit:            container.ResourceLimits().MemoryLimit(),
			EnvVars:                envVars,
			Secrets:                secrets,
			Networks:               networks,
			Mounts:                 mounts,
		}

		output.Containers = append(output.Containers, containerOutput)
	}

	return output, nil
}
