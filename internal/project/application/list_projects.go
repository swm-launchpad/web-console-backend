package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type ListProjectsInput struct {
	UserID uint
}

type ProjectListItem struct {
	ProjectID    uint   `json:"project_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	FQDN         string `json:"fqdn,omitempty"`
	Plan         string `json:"plan,omitempty"`
	Status       string `json:"status"`
	CPULimit     uint32 `json:"cpu_limit"`
	MemoryLimit  uint32 `json:"memory_limit"`
	DiskLimit    uint32 `json:"disk_limit"`
	TrafficLimit uint32 `json:"traffic_limit"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ListProjectsOutput struct {
	Projects []ProjectListItem `json:"projects"`
	Total    int               `json:"total"`
}

type ListProjectsUseCase struct {
	projectService service.ProjectService
	logger         logger.Logger
}

func NewListProjectsUseCase(projectService service.ProjectService, log logger.Logger) *ListProjectsUseCase {
	return &ListProjectsUseCase{
		projectService: projectService,
		logger:         log,
	}
}

func (uc *ListProjectsUseCase) Execute(ctx context.Context, input ListProjectsInput) (*ListProjectsOutput, error) {
	uc.logger.Info(ctx, "list projects started",
		zap.Uint("user_id", input.UserID),
	)

	// Get projects for the user
	projects, err := uc.projectService.ListProjects(ctx, input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to list projects",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Build output
	output := &ListProjectsOutput{
		Projects: make([]ProjectListItem, 0, len(projects)),
		Total:    len(projects),
	}

	for _, project := range projects {
		item := ProjectListItem{
			ProjectID:    project.ProjectID(),
			Name:         project.Name(),
			Slug:         project.Slug().String(),
			Status:       string(project.Status()),
			CPULimit:     project.Limits().CPULimit(),
			MemoryLimit:  project.Limits().MemoryLimit(),
			DiskLimit:    project.Limits().DiskLimit(),
			TrafficLimit: project.Limits().TrafficLimit(),
			CreatedAt:    project.CreatedAt().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    project.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if fqdn, ok := project.FQDN(); ok {
			item.FQDN = fqdn
		}

		if plan, ok := project.Plan(); ok {
			item.Plan = plan.String()
		}

		output.Projects = append(output.Projects, item)
	}

	uc.logger.Info(ctx, "list projects completed",
		zap.Uint("user_id", input.UserID),
		zap.Int("total_projects", len(projects)),
	)

	return output, nil
}
