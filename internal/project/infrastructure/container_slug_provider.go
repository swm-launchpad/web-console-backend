package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containerapp "github.com/swm-launchpad/web-console-backend/internal/container/application"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"go.uber.org/zap"
)

// containerSlugProvider implements ContainerSlugProvider by delegating to Container BC's application layer.
// This follows clean architecture by crossing bounded context boundaries through the application layer.
type containerSlugProvider struct {
	getContainerSlugsUC *containerapp.GetContainerSlugsByProjectIDUseCase
	logger              logger.Logger
}

// NewContainerSlugProvider creates a new ContainerSlugProvider implementation
func NewContainerSlugProvider(
	getContainerSlugsUC *containerapp.GetContainerSlugsByProjectIDUseCase,
	log logger.Logger,
) projectinfra.ContainerSlugProvider {
	return &containerSlugProvider{
		getContainerSlugsUC: getContainerSlugsUC,
		logger:              log,
	}
}

// GetContainerSlugsByProjectID retrieves container slugs from Container BC
func (p *containerSlugProvider) GetContainerSlugsByProjectID(ctx context.Context, projectID uint) ([]string, error) {
	p.logger.Debug(ctx, "container slug provider getting slugs",
		zap.Uint("project_id", projectID),
	)

	output, err := p.getContainerSlugsUC.Execute(ctx, containerapp.GetContainerSlugsByProjectIDInput{
		ProjectID: projectID,
	})
	if err != nil {
		p.logger.Error(ctx, "container slug provider failed to get slugs",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	p.logger.Debug(ctx, "container slug provider got slugs",
		zap.Uint("project_id", projectID),
		zap.Int("count", len(output.ContainerSlugs)),
	)

	return output.ContainerSlugs, nil
}
