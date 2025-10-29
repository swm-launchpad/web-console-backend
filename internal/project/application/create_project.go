package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type CreateProjectInput struct {
	Name         string
	OwnerID      uint
	FQDN         *string
	Plan         *value.Plan
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
	logger         logger.Logger
}

func NewCreateProjectUseCase(projectService service.ProjectService, txManager db.TxManager, log logger.Logger) *CreateProjectUseCase {
	return &CreateProjectUseCase{
		projectService: projectService,
		txManager:      txManager,
		logger:         log,
	}
}

func (uc *CreateProjectUseCase) Execute(ctx context.Context, input CreateProjectInput) (*CreateProjectOutput, error) {
	uc.logger.Info(ctx, "create project started",
		zap.String("name", input.Name),
		zap.Uint("owner_id", input.OwnerID),
		zap.Uint32("cpu_limit", input.CPULimit),
		zap.Uint32("memory_limit", input.MemoryLimit),
	)

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
			uc.logger.Error(ctx, "failed to create resource limits",
				zap.Error(err),
				zap.Uint("owner_id", input.OwnerID),
			)
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
			uc.logger.Error(ctx, "failed to create project",
				zap.Error(err),
				zap.String("name", input.Name),
				zap.Uint("owner_id", input.OwnerID),
			)
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
			plan = p.String()
			hasPlan = true
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "create project completed",
		zap.Uint("project_id", projectID),
		zap.String("name", name),
		zap.String("slug", slug),
	)

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
