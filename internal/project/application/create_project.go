package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type CreateProjectInput struct {
	Name          string
	Slug          string
	OwnerID       uint
	FQDN          *string
	Plan          *string
	CPULimit      *uint32
	MemoryRequest *uint32
	MemoryLimit   *uint32
	DiskLimit     *uint32
	TrafficLimit  *uint64
}

type CreateProjectOutput struct {
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	FQDN      string `json:"fqdn,omitempty"`
	Plan      string `json:"plan,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type CreateProjectUseCase struct {
	projectService service.ProjectService
	txManager      db.TxManager
}

func NewCreateProjectUseCase(projectService service.ProjectService, txManager db.TxManager) *CreateProjectUseCase {
	return &CreateProjectUseCase{
		projectService: projectService,
		txManager:      txManager,
	}
}

func (uc *CreateProjectUseCase) Execute(ctx context.Context, input CreateProjectInput) (*CreateProjectOutput, error) {
	var output *CreateProjectOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Create project slug
		slug, err := model.NewProjectSlug(input.Slug)
		if err != nil {
			return err
		}

		// Create project through service
		project, err := uc.projectService.CreateProject(txCtx, input.Name, *slug, input.OwnerID)
		if err != nil {
			return err
		}

		// Set optional fields
		if input.FQDN != nil {
			if err := project.SetFQDN(*input.FQDN); err != nil {
				return err
			}
		}

		if input.Plan != nil {
			if err := project.UpdatePlan(*input.Plan); err != nil {
				return err
			}
		}

		// TODO: 플랜에 따라, 리소스 제한량의 기본값을 설정해야 함
		// TODO: 플랜 구현시 이 부분을 함께 구현해야 함

		// Set resource limits if provided
		if input.CPULimit != nil || input.MemoryRequest != nil || input.MemoryLimit != nil || input.DiskLimit != nil || input.TrafficLimit != nil {
			limits, err := model.NewResourceLimits(
				input.CPULimit,
				input.MemoryRequest,
				input.MemoryLimit,
				input.DiskLimit,
				input.TrafficLimit,
			)
			if err != nil {
				return err
			}
			if err := project.UpdateResourceLimits(*limits); err != nil {
				return err
			}
		}

		// Build output
		output = &CreateProjectOutput{
			ProjectID: project.GetProjectID(),
			Name:      project.GetName(),
			Slug:      project.GetSlug().String(),
			Status:    string(project.GetStatus()),
			CreatedAt: project.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		}

		if project.GetFQDN() != nil {
			output.FQDN = *project.GetFQDN()
		}

		if project.GetPlan() != nil {
			output.Plan = *project.GetPlan()
		}

		return nil
	})

	return output, err
}
