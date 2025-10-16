package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type UpdateContainerInput struct {
	ContainerID    uint
	UserID         uint
	Name           *string
	StableWindow   *uint32
	GitURL         *string
	GitBranch      *string
	GitDirectory   *string
	CPULimit       *uint32
	MemoryLimit    *uint32
	TemplateID     *uint
	TemplateConfig map[string]interface{}
}

type UpdateContainerOutput struct {
	ContainerID uint   `json:"container_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateContainerUseCase struct {
	containerRepo         repository.ContainerRepository
	permissionSvc         service.PermissionService
	resourceValidationSvc service.ResourceValidationService
	txManager             db.TxManager
}

func NewUpdateContainerUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	resourceValidationSvc service.ResourceValidationService,
	txManager db.TxManager,
) *UpdateContainerUseCase {
	return &UpdateContainerUseCase{
		containerRepo:         containerRepo,
		permissionSvc:         permissionSvc,
		resourceValidationSvc: resourceValidationSvc,
		txManager:             txManager,
	}
}

func (uc *UpdateContainerUseCase) Execute(ctx context.Context, input UpdateContainerInput) (*UpdateContainerOutput, error) {
	var containerID uint
	var name, slug, updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			return err
		}

		// Get container with lock
		container, err := uc.containerRepo.FindByIDForUpdate(txCtx, input.ContainerID)
		if err != nil {
			return err
		}

		// Update name if provided
		if input.Name != nil {
			if err := container.ChangeName(*input.Name); err != nil {
				return err
			}
		}

		// Update stable window if provided
		if input.StableWindow != nil {
			container.SetStableWindow(input.StableWindow)
		}

		// Update git config if provided
		if input.GitURL != nil || input.GitBranch != nil || input.GitDirectory != nil {
			gitURL := container.GitConfig().RepositoryURL()
			gitBranch := container.GitConfig().Branch()
			gitDir := container.GitConfig().DirectoryPath()

			if input.GitURL != nil {
				gitURL = *input.GitURL
			}
			if input.GitBranch != nil {
				gitBranch = *input.GitBranch
			}
			if input.GitDirectory != nil {
				gitDir = input.GitDirectory
			}

			gitConfig, err := value.NewGitConfig(gitURL, gitBranch, gitDir)
			if err != nil {
				return err
			}

			if err := container.UpdateGitConfig(gitConfig); err != nil {
				return err
			}
		}

		// Update resource limits if provided
		if input.CPULimit != nil || input.MemoryLimit != nil {
			cpuLimit := container.ResourceLimits().CPULimit()
			memLimit := container.ResourceLimits().MemoryLimit()

			if input.CPULimit != nil {
				cpuLimit = input.CPULimit
			}
			if input.MemoryLimit != nil {
				memLimit = input.MemoryLimit
			}

			// Validate project resource limits before updating
			// Use 0 as default if limit is nil
			var cpuVal, memVal uint32
			if cpuLimit != nil {
				cpuVal = *cpuLimit
			}
			if memLimit != nil {
				memVal = *memLimit
			}

			if err := uc.resourceValidationSvc.ValidateProjectResourceLimits(
				txCtx,
				container.ProjectID(),
				cpuVal,
				memVal,
				input.ContainerID, // exclude this container from total calculation
			); err != nil {
				return err
			}

			resourceLimits, err := value.NewResourceLimits(cpuLimit, memLimit)
			if err != nil {
				return err
			}

			if err := container.UpdateResourceLimits(resourceLimits); err != nil {
				return err
			}
		}

		// Update template config if provided
		if input.TemplateConfig != nil {
			if err := container.UpdateTemplateConfig(input.TemplateID, input.TemplateConfig); err != nil {
				return err
			}
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		containerID = container.ContainerID()
		name = container.Name()
		slug = container.Slug().String()
		updatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &UpdateContainerOutput{
		ContainerID: containerID,
		Name:        name,
		Slug:        slug,
		UpdatedAt:   updatedAt,
	}, nil
}
