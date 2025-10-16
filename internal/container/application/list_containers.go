package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type ListContainersInput struct {
	ProjectID uint
	UserID    uint
}

type ContainerListItem struct {
	ContainerID uint    `json:"container_id"`
	ProjectID   uint    `json:"project_id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	FQDN        string  `json:"fqdn,omitempty"`
	GitURL      string  `json:"git_url"`
	GitBranch   string  `json:"git_branch"`
	CPULimit    *uint32 `json:"cpu_limit,omitempty"`    // Millicores (1000 = 1 CPU core)
	MemoryLimit *uint32 `json:"memory_limit,omitempty"` // Mi (Mebibytes)
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

type ListContainersOutput struct {
	Containers []ContainerListItem `json:"containers"`
	Total      int64               `json:"total"`
}

type ListContainersUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
}

func NewListContainersUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
) *ListContainersUseCase {
	return &ListContainersUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
	}
}

func (uc *ListContainersUseCase) Execute(ctx context.Context, input ListContainersInput) (*ListContainersOutput, error) {
	// Check permission to access project (using CreateContainer permission as proxy for project access)
	if err := uc.permissionSvc.CanUserCreateContainer(ctx, input.UserID, input.ProjectID); err != nil {
		return nil, err
	}

	// Get containers for project
	containers, err := uc.containerRepo.FindByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Count total
	total, err := uc.containerRepo.CountByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build output
	items := make([]ContainerListItem, 0, len(containers))
	for _, container := range containers {
		item := ContainerListItem{
			ContainerID: container.ContainerID(),
			ProjectID:   container.ProjectID(),
			Name:        container.Name(),
			Slug:        container.Slug().String(),
			GitURL:      container.GitConfig().RepositoryURL(),
			GitBranch:   container.GitConfig().Branch(),
			CPULimit:    container.ResourceLimits().CPULimit(),
			MemoryLimit: container.ResourceLimits().MemoryLimit(),
			CreatedAt:   container.CreatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if !container.UpdatedAt().IsZero() {
			item.UpdatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		items = append(items, item)
	}

	return &ListContainersOutput{
		Containers: items,
		Total:      total,
	}, nil
}
