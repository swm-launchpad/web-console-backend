package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type UpdateProjectInput struct {
	ProjectID     uint
	Name          *string
	FQDN          *string
	Plan          *string
	Status        *string
	CPULimit      *uint32
	MemoryRequest *uint32
	MemoryLimit   *uint32
	DiskLimit     *uint32
	TrafficLimit  *uint64
}

type UpdateProjectOutput struct {
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	FQDN      string `json:"fqdn,omitempty"`
	Plan      string `json:"plan,omitempty"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type UpdateProjectUseCase struct {
	projectService service.ProjectService
	txManager      db.TxManager
}

func NewUpdateProjectUseCase(projectService service.ProjectService, txManager db.TxManager) *UpdateProjectUseCase {
	return &UpdateProjectUseCase{
		projectService: projectService,
		txManager:      txManager,
	}
}

func (uc *UpdateProjectUseCase) Execute(ctx context.Context, input UpdateProjectInput) (*UpdateProjectOutput, error) {
	var output *UpdateProjectOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Update project through service
		project, err := uc.projectService.UpdateProject(txCtx, input.ProjectID, func(p *model.Project) error {
			// Update fields if provided
			if input.Name != nil {
				if err := p.UpdateName(*input.Name); err != nil {
					return err
				}
			}

			if input.FQDN != nil {
				if err := p.SetFQDN(*input.FQDN); err != nil {
					return err
				}
			}

			if input.Plan != nil {
				if err := p.UpdatePlan(*input.Plan); err != nil {
					return err
				}
			}

			if input.Status != nil {
				status := model.ProjectStatus(*input.Status)
				if err := p.UpdateStatus(status); err != nil {
					return err
				}
			}

			// Update resource limits if any provided
			if input.CPULimit != nil || input.MemoryLimit != nil || input.DiskLimit != nil || input.TrafficLimit != nil {
				// Get current limits
				currentLimits := p.GetLimits()

				// Use provided values or keep current ones
				cpuLimit := input.CPULimit
				if cpuLimit == nil {
					cpuLimit = currentLimits.GetCPULimit()
				}

				memoryRequest := input.MemoryRequest
				if memoryRequest == nil {
					memoryRequest = currentLimits.GetMemoryRequest()
				}

				memLimit := input.MemoryLimit
				if memLimit == nil {
					memLimit = currentLimits.GetMemoryLimit()
				}

				diskLimit := input.DiskLimit
				if diskLimit == nil {
					diskLimit = currentLimits.GetDiskLimit()
				}

				trafficLimit := input.TrafficLimit
				if trafficLimit == nil {
					trafficLimit = currentLimits.GetTrafficLimit()
				}

				limits, err := model.NewResourceLimits(cpuLimit, memoryRequest, memLimit, diskLimit, trafficLimit)
				if err != nil {
					return err
				}

				if err := p.UpdateResourceLimits(*limits); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		// Build output
		output = &UpdateProjectOutput{
			ProjectID: project.GetProjectID(),
			Name:      project.GetName(),
			Slug:      project.GetSlug().String(),
			Status:    string(project.GetStatus()),
		}

		if project.GetFQDN() != nil {
			output.FQDN = *project.GetFQDN()
		}

		if project.GetPlan() != nil {
			output.Plan = *project.GetPlan()
		}

		if project.GetUpdatedAt() != nil {
			output.UpdatedAt = project.GetUpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	return output, err
}
