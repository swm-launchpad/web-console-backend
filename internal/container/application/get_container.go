package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type GetContainerInput struct {
	ContainerID uint
	UserID      uint
}

type EnvVarOutput struct {
	EnvVarID uint   `json:"env_var_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type NetworkOutput struct {
	NetworkID    uint   `json:"network_id"`
	InternalPort uint16 `json:"internal_port"`
	ExternalPort uint16 `json:"external_port,omitempty"`
	ExternalIP   string `json:"external_ip,omitempty"`
	FQDN         string `json:"fqdn,omitempty"`
	NetworkType  string `json:"network_type"`
}

type SecretOutput struct {
	SecretID uint   `json:"secret_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type BuildVarOutput struct {
	BuildVarID uint   `json:"build_var_id"`
	Key        string `json:"key"`
	Value      string `json:"value"`
}

type MountOutput struct {
	VolumeID  uint   `json:"volume_id"`
	MountPath string `json:"mount_path"`
}

type GetContainerOutput struct {
	ContainerID          uint                   `json:"container_id"`
	ProjectID            uint                   `json:"project_id"`
	TemplateID           uint                   `json:"template_id,omitempty"`
	Name                 string                 `json:"name"`
	Slug                 string                 `json:"slug"`
	StableWindow         uint32                 `json:"stable_window,omitempty"`
	TemplateConfig       map[string]interface{} `json:"template_config,omitempty"`
	GitURL               string                 `json:"git_url"`
	GitBranch            string                 `json:"git_branch"`
	GitSubpath           string                 `json:"git_subpath,omitempty"`
	GitHubInstallationID *int64                 `json:"github_installation_id,omitempty"`
	GitCommitHash        string                 `json:"git_commit_hash,omitempty"`
	LastBuiltCommit      string                 `json:"last_built_commit,omitempty"`
	CPULimit             uint32                 `json:"cpu_limit"`
	MemoryLimit          uint32                 `json:"memory_limit"`
	MonthlyBuildTime     uint32                 `json:"monthly_build_time,omitempty"`
	MonthlyBuildCount    uint32                 `json:"monthly_build_count,omitempty"`
	MonthlyUptime        string                 `json:"monthly_uptime,omitempty"`
	EnvVars              []EnvVarOutput         `json:"env_vars"`
	Networks             []NetworkOutput        `json:"networks"`
	Secrets              []SecretOutput         `json:"secrets"`
	BuildVars            []BuildVarOutput       `json:"build_vars"`
	Mounts               []MountOutput          `json:"mounts"`
	CreatedAt            string                 `json:"created_at"`
	UpdatedAt            string                 `json:"updated_at,omitempty"`
}

type GetContainerUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	logger        logger.Logger
}

func NewGetContainerUseCase(containerRepo repository.ContainerRepository, permissionSvc service.PermissionService, log logger.Logger) *GetContainerUseCase {
	return &GetContainerUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		logger:        log,
	}
}

func (uc *GetContainerUseCase) Execute(ctx context.Context, input GetContainerInput) (*GetContainerOutput, error) {
	uc.logger.Info(ctx, "get container started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
	)

	// Get container
	container, err := uc.containerRepo.FindByID(ctx, input.ContainerID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find container",
			zap.Error(err),
			zap.Uint("container_id", input.ContainerID),
		)
		return nil, err
	}

	// Check permission
	if err := uc.permissionSvc.CanUserAccessContainer(ctx, input.UserID, input.ContainerID); err != nil {
		uc.logger.Warn(ctx, "permission check failed",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
		)
		return nil, err
	}

	// Build output
	output := &GetContainerOutput{
		ContainerID: container.ContainerID(),
		ProjectID:   container.ProjectID(),
		Name:        container.Name(),
		Slug:        container.Slug().String(),
		GitURL:      container.GitConfig().RepositoryURL(),
		GitBranch:   container.GitConfig().Branch(),
		CPULimit:    container.ResourceLimits().CPULimitOrDefault(0),
		MemoryLimit: container.ResourceLimits().MemoryLimitOrDefault(0),
		EnvVars:     make([]EnvVarOutput, 0),
		Networks:    make([]NetworkOutput, 0),
		Secrets:     make([]SecretOutput, 0),
		BuildVars:   make([]BuildVarOutput, 0),
		Mounts:      make([]MountOutput, 0),
		CreatedAt:   container.CreatedAt().Format("2006-01-02T15:04:05Z"),
	}

	if !container.UpdatedAt().IsZero() {
		output.UpdatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")
	}

	if templateID := container.TemplateID(); templateID != nil {
		output.TemplateID = *templateID
	}

	if stableWindow := container.StableWindow(); stableWindow != nil {
		output.StableWindow = *stableWindow
	}

	if config := container.TemplateConfig(); config != nil {
		output.TemplateConfig = config
	}

	if gitDir := container.GitConfig().DirectoryPath(); gitDir != nil {
		output.GitSubpath = *gitDir
	}

	if installationID := container.GitHubInstallationID(); installationID != nil {
		output.GitHubInstallationID = installationID
	}

	if gitCommit := container.GitCommitHash(); gitCommit != nil {
		output.GitCommitHash = *gitCommit
	}

	if lastBuilt := container.LastBuiltGitCommitHash(); lastBuilt != nil {
		output.LastBuiltCommit = *lastBuilt
	}

	if buildTime := container.MonthlyBuildTime(); buildTime != nil {
		output.MonthlyBuildTime = *buildTime
	}

	if buildCount := container.MonthlyBuildCount(); buildCount != nil {
		output.MonthlyBuildCount = *buildCount
	}

	if uptime := container.MonthlyUptime(); uptime != nil {
		output.MonthlyUptime = *uptime
	}

	// Map environment variables
	for _, envVar := range container.EnvVars() {
		output.EnvVars = append(output.EnvVars, EnvVarOutput{
			EnvVarID: envVar.EnvVarID(),
			Key:      envVar.Key(),
			Value:    envVar.Value(),
		})
	}

	// Map networks
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

		output.Networks = append(output.Networks, netOutput)
	}

	// Map secrets
	for _, secret := range container.Secrets() {
		output.Secrets = append(output.Secrets, SecretOutput{
			SecretID: secret.SecretID(),
			Key:      secret.Key(),
			Value:    secret.Value(),
		})
	}

	// Map build variables
	for _, buildVar := range container.BuildVars() {
		output.BuildVars = append(output.BuildVars, BuildVarOutput{
			BuildVarID: buildVar.BuildVarID(),
			Key:        buildVar.Key(),
			Value:      buildVar.Value(),
		})
	}

	// Map mounts
	for _, mount := range container.Mounts() {
		output.Mounts = append(output.Mounts, MountOutput{
			VolumeID:  mount.VolumeID(),
			MountPath: mount.MountPath(),
		})
	}

	uc.logger.Info(ctx, "get container completed",
		zap.Uint("container_id", container.ContainerID()),
		zap.String("name", container.Name()),
	)

	return output, nil
}
