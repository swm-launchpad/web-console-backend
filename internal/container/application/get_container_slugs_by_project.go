package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

// GetContainerSlugsByProjectIDInput defines the input for getting container slugs
type GetContainerSlugsByProjectIDInput struct {
	ProjectID uint
}

// GetContainerSlugsByProjectIDOutput defines the output containing container slugs
type GetContainerSlugsByProjectIDOutput struct {
	ContainerSlugs []string `json:"container_slugs"`
}

// GetContainerSlugsByProjectIDUseCase retrieves all container slugs for a project
// including soft-deleted containers. This is used for cleanup operations.
type GetContainerSlugsByProjectIDUseCase struct {
	containerRepo repository.ContainerRepository
	logger        logger.Logger
}

// NewGetContainerSlugsByProjectIDUseCase creates a new use case instance
func NewGetContainerSlugsByProjectIDUseCase(
	containerRepo repository.ContainerRepository,
	log logger.Logger,
) *GetContainerSlugsByProjectIDUseCase {
	return &GetContainerSlugsByProjectIDUseCase{
		containerRepo: containerRepo,
		logger:        log,
	}
}

// Execute retrieves all container slugs (including soft-deleted) for a project
func (uc *GetContainerSlugsByProjectIDUseCase) Execute(ctx context.Context, input GetContainerSlugsByProjectIDInput) (*GetContainerSlugsByProjectIDOutput, error) {
	uc.logger.Info(ctx, "get container slugs by project ID started",
		zap.Uint("project_id", input.ProjectID),
	)

	slugs, err := uc.containerRepo.FindAllSlugsByProjectIDIncludingDeleted(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get container slugs",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "get container slugs by project ID completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Int("count", len(slugs)),
	)

	return &GetContainerSlugsByProjectIDOutput{
		ContainerSlugs: slugs,
	}, nil
}
