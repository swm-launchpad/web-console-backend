package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type ListContainersInput struct {
	ProjectID uint
	UserID    uint
}

type ContainerListItem struct {
	ContainerID          uint    `json:"container_id"`
	ProjectID            uint    `json:"project_id"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	FQDN                 string  `json:"fqdn,omitempty"`
	GitURL               string  `json:"git_url"`
	GitBranch            string  `json:"git_branch"`
	GitHubInstallationID *int64  `json:"github_installation_id,omitempty"`
	CPULimit             *uint32 `json:"cpu_limit,omitempty"`    // Millicores (1000 = 1 CPU core)
	MemoryLimit          *uint32 `json:"memory_limit,omitempty"` // Mi (Mebibytes)
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at,omitempty"`
}

type ListContainersOutput struct {
	Containers []ContainerListItem `json:"containers"`
	Total      int64               `json:"total"`
}

type ListContainersUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	logger        logger.Logger
}

func NewListContainersUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	log logger.Logger,
) *ListContainersUseCase {
	return &ListContainersUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		logger:        log,
	}
}

func (uc *ListContainersUseCase) Execute(ctx context.Context, input ListContainersInput) (*ListContainersOutput, error) {
	uc.logger.Info(ctx, "list containers started",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint("user_id", input.UserID),
	)

	// Check permission to access project (using CreateContainer permission as proxy for project access)
	if err := uc.permissionSvc.CanUserCreateContainer(ctx, input.UserID, input.ProjectID); err != nil {
		uc.logger.Error(ctx, "permission check failed",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Get containers for project
	containers, err := uc.containerRepo.FindByProjectID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find containers",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Count total
	total, err := uc.containerRepo.CountByProjectID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to count containers",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	items := make([]ContainerListItem, 0, len(containers))
	for _, container := range containers {
		item := ContainerListItem{
			ContainerID:          container.ContainerID(),
			ProjectID:            container.ProjectID(),
			Name:                 container.Name(),
			Slug:                 container.Slug().String(),
			GitURL:               container.GitConfig().RepositoryURL(),
			GitBranch:            container.GitConfig().Branch(),
			GitHubInstallationID: container.GitHubInstallationID(),
			CPULimit:             container.ResourceLimits().CPULimit(),
			MemoryLimit:          container.ResourceLimits().MemoryLimit(),
			CreatedAt:            container.CreatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if !container.UpdatedAt().IsZero() {
			item.UpdatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		items = append(items, item)
	}

	uc.logger.Info(ctx, "list containers completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Int64("total", total),
	)

	return &ListContainersOutput{
		Containers: items,
		Total:      total,
	}, nil
}
