package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type CreateProjectInput struct {
	Name         string
	OwnerID      uint
	FQDN         *string
	Plan         *string
	CPULimit     uint32
	MemoryLimit  uint32
	DiskLimit    uint32
	TrafficLimit uint32
}

type CreateProjectOutput struct {
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
	UpdatedAt    string `json:"updated_at,omitempty"`
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
	var projectID uint
	var name string
	var slug string
	var status string
	var cpuLimit, memoryLimit, diskLimit, trafficLimit uint32
	var fqdn, plan string
	var hasFQDN, hasPlan bool
	var createdAt, updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Prepare resource limits (all required)
		limits, err := value.NewResourceLimits(
			input.CPULimit,
			input.MemoryLimit,
			input.DiskLimit,
			input.TrafficLimit,
		)
		if err != nil {
			return err
		}

		// Create project through service
		// Slug is automatically generated from name by the service
		project, err := uc.projectService.CreateProject(
			txCtx,
			input.Name,
			input.OwnerID,
			*limits,
			input.FQDN,
			input.Plan,
		)
		if err != nil {
			return err
		}

		// Extract primitive values within transaction
		projectID = project.ProjectID()
		name = project.Name()
		slug = project.Slug().String()
		status = string(project.Status())
		cpuLimit = project.Limits().CPULimit()
		memoryLimit = project.Limits().MemoryLimit()
		diskLimit = project.Limits().DiskLimit()
		trafficLimit = project.Limits().TrafficLimit()
		createdAt = project.CreatedAt().Format("2006-01-02T15:04:05Z")
		updatedAt = project.UpdatedAt().Format("2006-01-02T15:04:05Z")

		if f, ok := project.FQDN(); ok {
			fqdn = f
			hasFQDN = true
		}

		if p, ok := project.Plan(); ok {
			plan = p
			hasPlan = true
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build output after successful transaction
	output := &CreateProjectOutput{
		ProjectID:    projectID,
		Name:         name,
		Slug:         slug,
		Status:       status,
		CPULimit:     cpuLimit,
		MemoryLimit:  memoryLimit,
		DiskLimit:    diskLimit,
		TrafficLimit: trafficLimit,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	if hasFQDN {
		output.FQDN = fqdn
	}

	if hasPlan {
		output.Plan = plan
	}

	return output, nil
}
