package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/deploy"
	"go.uber.org/zap"
)

type UpdateProjectInput struct {
	ProjectID    uint
	ActingUserID uint // The user performing the update (for quota validation)
	Name         *string
	Plan         *value.Plan
	Status       *string
	CPULimit     *uint32
	MemoryLimit  *uint32
	DiskLimit    *uint32
	TrafficLimit *uint32
}

type UpdateProjectOutput struct {
	ProjectID    uint   `json:"project_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Plan         string `json:"plan,omitempty"`
	Status       string `json:"status"`
	CPULimit     uint32 `json:"cpu_limit"`
	MemoryLimit  uint32 `json:"memory_limit"`
	DiskLimit    uint32 `json:"disk_limit"`
	TrafficLimit uint32 `json:"traffic_limit"`
	UpdatedAt    string `json:"updated_at"`
}

type UpdateProjectUseCase struct {
	projectService service.ProjectService
	deployService  deploy.Deployer
	txManager      db.TxManager
	logger         logger.Logger
}

func NewUpdateProjectUseCase(projectService service.ProjectService, deployService deploy.Deployer, txManager db.TxManager, log logger.Logger) *UpdateProjectUseCase {
	return &UpdateProjectUseCase{
		projectService: projectService,
		deployService:  deployService,
		txManager:      txManager,
		logger:         log,
	}
}

func (uc *UpdateProjectUseCase) Execute(ctx context.Context, input UpdateProjectInput) (*UpdateProjectOutput, error) {
	uc.logger.Info(ctx, "update project started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Check if plan is changing (for triggering redeployment)
	// This check is done before the transaction to avoid blocking the transaction
	// If this fails, we log the error but don't fail the update operation
	var planChanged bool
	if input.Plan != nil {
		// Get current project to check plan change
		currentProject, err := uc.projectService.GetProject(ctx, input.ProjectID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get current project for plan change detection",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			// Don't fail the update operation, just skip redeployment trigger
			// planChanged remains false
		} else {
			if currentPlan, ok := currentProject.Plan(); ok {
				planChanged = currentPlan.String() != input.Plan.String()
			} else {
				// No current plan, so setting a plan is considered a change
				planChanged = true
			}
		}
	}

	var projectID uint
	var name string
	var slug string
	var status string
	var plan string
	var hasPlan bool
	var cpuLimit, memoryLimit, diskLimit, trafficLimit uint32
	var updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Update project through service
		project, err := uc.projectService.UpdateProject(txCtx, input.ProjectID, input.ActingUserID, func(p *model.Project) error {
			// Update fields if provided
			if input.Name != nil {
				if err := p.SetName(*input.Name); err != nil {
					return err
				}
			}

			if input.Plan != nil {
				if err := p.SetPlan(*input.Plan); err != nil {
					return err
				}
			}

			if input.Status != nil {
				status := value.ProjectStatus(*input.Status)
				if err := p.SetStatus(status); err != nil {
					return err
				}
			}

			// Update resource limits if any provided
			if input.CPULimit != nil || input.MemoryLimit != nil || input.DiskLimit != nil || input.TrafficLimit != nil {
				// Get current limits
				currentLimits := p.Limits()

				// Use provided values or keep current ones (nil = keep current, non-nil = update)
				cpu := currentLimits.CPULimit()
				if input.CPULimit != nil {
					cpu = *input.CPULimit
				}

				memory := currentLimits.MemoryLimit()
				if input.MemoryLimit != nil {
					memory = *input.MemoryLimit
				}

				disk := currentLimits.DiskLimit()
				if input.DiskLimit != nil {
					disk = *input.DiskLimit
				}

				traffic := currentLimits.TrafficLimit()
				if input.TrafficLimit != nil {
					traffic = *input.TrafficLimit
				}

				limits, err := value.NewResourceLimits(cpu, memory, disk, traffic)
				if err != nil {
					return err
				}

				if err := p.SetResourceLimits(*limits); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			uc.logger.Error(ctx, "failed to update project",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Extract primitive values within transaction
		projectID = project.ProjectID()
		name = project.Name()
		slug = project.Slug().String()
		status = string(project.Status())
		updatedAt = project.UpdatedAt().Format("2006-01-02T15:04:05Z")

		// Extract resource limits
		limits := project.Limits()
		cpuLimit = limits.CPULimit()
		memoryLimit = limits.MemoryLimit()
		diskLimit = limits.DiskLimit()
		trafficLimit = limits.TrafficLimit()

		if p, ok := project.Plan(); ok {
			plan = p.String()
			hasPlan = true
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "update project completed",
		zap.Uint("project_id", projectID),
		zap.String("name", name),
	)

	// Trigger redeployment if plan changed and deployService is available
	if planChanged && uc.deployService != nil {
		uc.logger.Info(ctx, "plan changed, triggering redeployment",
			zap.Uint("project_id", projectID),
		)

		// Trigger build and deploy in background (fire and forget)
		// This is async and should not block the response
		go func() {
			// Use background context to prevent cancellation
			bgCtx := context.Background()
			if err := uc.deployService.BuildAndDeployProject(bgCtx, projectID); err != nil {
				uc.logger.Error(bgCtx, "failed to trigger redeployment after plan change",
					zap.Error(err),
					zap.Uint("project_id", projectID),
				)
			} else {
				uc.logger.Info(bgCtx, "redeployment triggered successfully after plan change",
					zap.Uint("project_id", projectID),
				)
			}
		}()
	}

	// Build output after successful transaction
	output := &UpdateProjectOutput{
		ProjectID:    projectID,
		Name:         name,
		Slug:         slug,
		Status:       status,
		CPULimit:     cpuLimit,
		MemoryLimit:  memoryLimit,
		DiskLimit:    diskLimit,
		TrafficLimit: trafficLimit,
		UpdatedAt:    updatedAt,
	}

	if hasPlan {
		output.Plan = plan
	}

	return output, nil
}
