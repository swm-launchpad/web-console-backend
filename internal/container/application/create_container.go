package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectservice "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	userrepository "github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type VolumeToCreate struct {
	Name      string
	Capacity  uint32
	MountPath string
}

type CreateContainerInput struct {
	ProjectID            uint
	UserID               uint
	TemplateID           *uint
	Name                 string
	GitURL               string
	GitBranch            string
	GitDirectory         *string
	GitHubInstallationID *int64 // Optional: for private repositories
	CPULimit             uint32
	MemoryLimit          uint32
	TemplateConfig       map[string]interface{}
	Volumes              []VolumeToCreate
}

type CreateContainerOutput struct {
	ContainerID  uint   `json:"container_id"`
	ProjectID    uint   `json:"project_id"`
	TemplateID   uint   `json:"template_id,omitempty"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	GitURL       string `json:"git_url"`
	GitBranch    string `json:"git_branch"`
	GitDirectory string `json:"git_directory,omitempty"`
	CPULimit     uint32 `json:"cpu_limit"`
	MemoryLimit  uint32 `json:"memory_limit"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type CreateContainerUseCase struct {
	containerService      service.ContainerService
	containerRepo         repository.ContainerRepository
	permissionSvc         service.PermissionService
	resourceValidationSvc service.ResourceValidationService
	volumeService         projectservice.VolumeService
	installationRepo      userrepository.GitHubInstallationRepository
	txManager             db.TxManager
	logger                logger.Logger
}

func NewCreateContainerUseCase(
	containerService service.ContainerService,
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	resourceValidationSvc service.ResourceValidationService,
	volumeService projectservice.VolumeService,
	installationRepo userrepository.GitHubInstallationRepository,
	txManager db.TxManager,
	log logger.Logger,
) *CreateContainerUseCase {
	return &CreateContainerUseCase{
		containerService:      containerService,
		containerRepo:         containerRepo,
		permissionSvc:         permissionSvc,
		resourceValidationSvc: resourceValidationSvc,
		volumeService:         volumeService,
		installationRepo:      installationRepo,
		txManager:             txManager,
		logger:                log,
	}
}

func (uc *CreateContainerUseCase) Execute(ctx context.Context, input CreateContainerInput) (*CreateContainerOutput, error) {
	uc.logger.Info(ctx, "create container started",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint("user_id", input.UserID),
		zap.String("name", input.Name),
		zap.Uint32("cpu_limit", input.CPULimit),
		zap.Uint32("memory_limit", input.MemoryLimit),
	)

	var containerID, projectID uint
	var templateID *uint
	var name, slug, gitURL, gitBranch string
	var gitDirectory *string
	var cpuLimit, memoryLimit uint32
	var createdAt, updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserCreateContainer(txCtx, input.UserID, input.ProjectID); err != nil {
			uc.logger.Error(ctx, "permission check failed",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Validate GitHub installation ownership if provided
		if input.GitHubInstallationID != nil {
			if err := uc.installationRepo.ValidateUserOwnership(txCtx, *input.GitHubInstallationID, input.UserID); err != nil {
				uc.logger.Error(ctx, "GitHub installation ownership validation failed",
					zap.Error(err),
					zap.Int64("installation_id", *input.GitHubInstallationID),
				)
				return err
			}
		}

		// Validate project resource limits
		if err := uc.resourceValidationSvc.ValidateProjectResourceLimits(
			txCtx,
			input.ProjectID,
			input.CPULimit,
			input.MemoryLimit,
			0, // excludeContainerID = 0 for new containers
		); err != nil {
			uc.logger.Error(ctx, "resource limits validation failed",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Create Git configuration
		gitConfig, err := value.NewGitConfig(input.GitURL, input.GitBranch, input.GitDirectory)
		if err != nil {
			uc.logger.Error(ctx, "failed to create git config",
				zap.Error(err),
			)
			return err
		}

		// Create resource limits
		cpuLimitPtr := &input.CPULimit
		memLimitPtr := &input.MemoryLimit
		resourceLimits, err := value.NewResourceLimits(cpuLimitPtr, memLimitPtr)
		if err != nil {
			uc.logger.Error(ctx, "failed to create resource limits",
				zap.Error(err),
			)
			return err
		}

		// Create container through service
		// Slug is automatically generated from name by the service
		container, err := uc.containerService.CreateContainer(
			txCtx,
			input.ProjectID,
			input.Name,
			gitConfig,
			resourceLimits,
			input.TemplateID,
			input.TemplateConfig,
			input.GitHubInstallationID,
		)
		if err != nil {
			uc.logger.Error(ctx, "failed to create container",
				zap.Error(err),
				zap.String("name", input.Name),
			)
			return err
		}

		// Create volumes and add mounts if volumes are specified
		if len(input.Volumes) > 0 {
			for _, volToCreate := range input.Volumes {
				// Create volume using VolumeService
				volume, err := uc.volumeService.CreateVolume(
					txCtx,
					input.ProjectID,
					volToCreate.Name,
					volToCreate.Capacity,
				)
				if err != nil {
					return err
				}

				// Add mount to container
				_, err = container.AddMount(volume.VolumeID(), volToCreate.MountPath)
				if err != nil {
					return err
				}
			}

			// Save container with mounts using repository
			if err := uc.containerRepo.Save(txCtx, container); err != nil {
				return err
			}
		}

		// Extract primitive values within transaction
		containerID = container.ContainerID()
		projectID = container.ProjectID()
		templateID = container.TemplateID()
		name = container.Name()
		slug = container.Slug().String()
		gitURL = container.GitConfig().RepositoryURL()
		gitBranch = container.GitConfig().Branch()
		gitDirectory = container.GitConfig().DirectoryPath()
		cpuLimit = container.ResourceLimits().CPULimitOrDefault(0)
		memoryLimit = container.ResourceLimits().MemoryLimitOrDefault(0)
		createdAt = container.CreatedAt().Format("2006-01-02T15:04:05Z")
		if !container.UpdatedAt().IsZero() {
			updatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "create container completed",
		zap.Uint("container_id", containerID),
		zap.Uint("project_id", projectID),
		zap.String("slug", slug),
	)

	// Build output after successful transaction
	output := &CreateContainerOutput{
		ContainerID: containerID,
		ProjectID:   projectID,
		Name:        name,
		Slug:        slug,
		GitURL:      gitURL,
		GitBranch:   gitBranch,
		CPULimit:    cpuLimit,
		MemoryLimit: memoryLimit,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if templateID != nil {
		output.TemplateID = *templateID
	}

	if gitDirectory != nil {
		output.GitDirectory = *gitDirectory
	}

	return output, nil
}
