package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type GetProjectBySlugInput struct {
	Slug string
}

type GetProjectBySlugUseCase struct {
	projectService service.ProjectService
	volumeService  service.VolumeService
	logger         logger.Logger
}

func NewGetProjectBySlugUseCase(projectService service.ProjectService, volumeService service.VolumeService, log logger.Logger) *GetProjectBySlugUseCase {
	return &GetProjectBySlugUseCase{
		projectService: projectService,
		volumeService:  volumeService,
		logger:         log,
	}
}

func (uc *GetProjectBySlugUseCase) Execute(ctx context.Context, input GetProjectBySlugInput) (*GetProjectOutput, error) {
	uc.logger.Info(ctx, "get project by slug started",
		zap.String("slug", input.Slug),
	)

	// Get project by slug
	project, err := uc.projectService.GetProjectBySlug(ctx, input.Slug)
	if err != nil {
		uc.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("slug", input.Slug),
		)
		return nil, err
	}

	// Use shared helper function to build output
	// This eliminates 75 lines of duplicated code
	output, err := buildProjectOutput(ctx, project, uc.volumeService)
	if err != nil {
		uc.logger.Error(ctx, "failed to build project output",
			zap.Error(err),
			zap.String("slug", input.Slug),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "get project by slug completed",
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", input.Slug),
	)

	return output, nil
}
