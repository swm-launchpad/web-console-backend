package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type ListProjectsInput struct {
	UserID *uint
	Offset int
	Limit  int
}

type ProjectListItem struct {
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	FQDN      string `json:"fqdn,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type ListProjectsOutput struct {
	Projects []ProjectListItem `json:"projects"`
	Total    int               `json:"total"`
}

type ListProjectsUseCase struct {
	projectService service.ProjectService
}

func NewListProjectsUseCase(projectService service.ProjectService) *ListProjectsUseCase {
	return &ListProjectsUseCase{
		projectService: projectService,
	}
}

func (uc *ListProjectsUseCase) Execute(ctx context.Context, input ListProjectsInput) (*ListProjectsOutput, error) {
	var projects []*model.Project
	var err error

	// Validate and set defaults
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	// Get projects based on user ID or all
	if input.UserID != nil {
		projects, err = uc.projectService.ListProjects(ctx, *input.UserID)
	} else {
		projects, err = uc.projectService.ListAllProjects(ctx, input.Offset, input.Limit)
	}

	if err != nil {
		return nil, err
	}

	// Build output
	output := &ListProjectsOutput{
		Projects: make([]ProjectListItem, 0, len(projects)),
		Total:    len(projects),
	}

	for _, project := range projects {
		item := ProjectListItem{
			ProjectID: project.GetProjectID(),
			Name:      project.GetName(),
			Slug:      project.GetSlug().String(),
			Status:    string(project.GetStatus()),
			CreatedAt: project.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if project.GetFQDN() != nil {
			item.FQDN = *project.GetFQDN()
		}

		output.Projects = append(output.Projects, item)
	}

	return output, nil
}
