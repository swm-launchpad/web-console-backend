package application

import (
	"context"
	"strings"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
	"go.uber.org/zap"
)

type ListProjectsInput struct {
	UserID uint
}

type ProjectSummary struct {
	ContainerCount  int      `json:"container_count"`
	DomainCount     int      `json:"domain_count"`
	Domains         []string `json:"domains,omitempty"`
	TotalCPUUsed    uint32   `json:"total_cpu_used"`
	TotalMemoryUsed uint32   `json:"total_memory_used"`
	TotalDiskUsed   uint32   `json:"total_disk_used"`
}

type ProjectListItem struct {
	ProjectID    uint            `json:"project_id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Plan         string          `json:"plan,omitempty"`
	Status       string          `json:"status"`
	CPULimit     uint32          `json:"cpu_limit"`
	MemoryLimit  uint32          `json:"memory_limit"`
	DiskLimit    uint32          `json:"disk_limit"`
	TrafficLimit uint32          `json:"traffic_limit"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
	Summary      *ProjectSummary `json:"summary,omitempty"`
}

type ListProjectsOutput struct {
	Projects []ProjectListItem `json:"projects"`
	Total    int               `json:"total"`
	Meta     *ListProjectsMeta `json:"meta,omitempty"`
}

type ListProjectsMeta struct {
	MaxProjectsPerUser int `json:"max_projects_per_user"`
}

type ListProjectsUseCase struct {
	projectService  service.ProjectService
	queries         *sqlc.Queries
	settingsService settings.SettingsService
	logger          logger.Logger
}

func NewListProjectsUseCase(projectService service.ProjectService, queries *sqlc.Queries, settingsService settings.SettingsService, log logger.Logger) *ListProjectsUseCase {
	return &ListProjectsUseCase{
		projectService:  projectService,
		queries:         queries,
		settingsService: settingsService,
		logger:          log,
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

	// Extract project IDs for summary query
	projectIDs := make([]uint32, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, uint32(project.ProjectID()))
	}

	// Get project summaries (container count, domain count, domains)
	var summaryMap map[uint32]*ProjectSummary
	if len(projectIDs) > 0 && uc.queries != nil {
		summaries, err := uc.queries.GetProjectsSummary(ctx, projectIDs)
		if err != nil {
			uc.logger.Error(ctx, "failed to get projects summary",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
			)
			// Don't fail the entire request, just log the error
			// Projects will be returned without summary
		} else {
			// Build summary map for fast lookup
			summaryMap = make(map[uint32]*ProjectSummary, len(summaries))
			for _, s := range summaries {
				var domains []string
				if s.Domains.Valid && s.Domains.String != "" {
					domains = strings.Split(s.Domains.String, ",")
				}

			// Convert interface{} to uint32 for resource fields
			var totalCPU, totalMemory, totalDisk uint32
			if v, ok := s.TotalCpuUsed.(int64); ok {
				totalCPU = uint32(v)
			}
			if v, ok := s.TotalMemoryUsed.(int64); ok {
				totalMemory = uint32(v)
			}
			if v, ok := s.TotalDiskUsed.(int64); ok {
				totalDisk = uint32(v)
			}
				summaryMap[s.ProjectID] = &ProjectSummary{
					ContainerCount:  int(s.ContainerCount),
					DomainCount:     int(s.DomainCount),
					Domains:         domains,
				TotalCPUUsed:    totalCPU,
				TotalMemoryUsed: totalMemory,
				TotalDiskUsed:   totalDisk,
				}
			}
		}
	}

	// Get max projects per user setting
	maxProjectsPerUser := 3 // default value
	if uc.settingsService != nil {
		if maxProjects, err := uc.settingsService.GetMaxProjectsPerUser(); err == nil {
			maxProjectsPerUser = maxProjects
		} else {
			uc.logger.Warn(ctx, "failed to get max_projects_per_user setting, using default",
				zap.Error(err),
				zap.Int("default_value", maxProjectsPerUser),
			)
		}
	}

	// Build output
	output := &ListProjectsOutput{
		Projects: make([]ProjectListItem, 0, len(projects)),
		Total:    len(projects),
		Meta: &ListProjectsMeta{
			MaxProjectsPerUser: maxProjectsPerUser,
		},
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

		if plan, ok := project.Plan(); ok {
			item.Plan = plan.String()
		}

		// Add summary if available
		if summaryMap != nil {
			if summary, ok := summaryMap[uint32(project.ProjectID())]; ok {
				item.Summary = summary
			}
		}

		output.Projects = append(output.Projects, item)
	}

	uc.logger.Info(ctx, "list projects completed",
		zap.Uint("user_id", input.UserID),
		zap.Int("total_projects", len(projects)),
	)

	return output, nil
}
