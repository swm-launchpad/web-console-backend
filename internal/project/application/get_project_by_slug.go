package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type GetProjectBySlugInput struct {
	Slug string
}

type GetProjectBySlugUseCase struct {
	projectService service.ProjectService
	volumeService  service.VolumeService
}

func NewGetProjectBySlugUseCase(projectService service.ProjectService, volumeService service.VolumeService) *GetProjectBySlugUseCase {
	return &GetProjectBySlugUseCase{
		projectService: projectService,
		volumeService:  volumeService,
	}
}

func (uc *GetProjectBySlugUseCase) Execute(ctx context.Context, input GetProjectBySlugInput) (*GetProjectOutput, error) {
	// Get project by slug
	project, err := uc.projectService.GetProjectBySlug(ctx, input.Slug)
	if err != nil {
		return nil, err
	}

	// Use shared helper function to build output
	// This eliminates 75 lines of duplicated code
	return buildProjectOutput(ctx, project, uc.volumeService)
}
