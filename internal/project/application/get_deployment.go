package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type GetDeploymentInput struct {
	ProjectID uint
}

type GetDeploymentOutput struct {
	DeploymentID          uint64 `json:"deployment_id"`
	ProjectID             uint   `json:"project_id"`
	Status                string `json:"status"`
	TektonEventID         string `json:"tekton_event_id,omitempty"`
	TektonPipelineRunName string `json:"tekton_pipeline_run_name,omitempty"`
	Summary               string `json:"summary,omitempty"`
	StartedAt             string `json:"started_at,omitempty"`
	FinishedAt            string `json:"finished_at,omitempty"`
	CreatedAt             string `json:"created_at"`
}

type GetDeploymentUseCase struct {
	deployService service.DeployService
	logger        logger.Logger
}

func NewGetDeploymentUseCase(
	deployService service.DeployService,
	log logger.Logger,
) *GetDeploymentUseCase {
	return &GetDeploymentUseCase{
		deployService: deployService,
		logger:        log,
	}
}

func (uc *GetDeploymentUseCase) Execute(ctx context.Context, input GetDeploymentInput) (*GetDeploymentOutput, error) {
	uc.logger.Info(ctx, "get deployment started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Note: Permission check is performed in the handler to prevent information disclosure
	// The handler converts permission errors to "not found" errors

	// Get latest deployment status from database (lightweight operation)
	deployment, err := uc.deployService.GetDeploymentStatus(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get deployment status",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	output := &GetDeploymentOutput{
		DeploymentID: uint64(deployment.DeploymentID),
		ProjectID:    uint(deployment.ProjectID()),
		Status:       string(deployment.Status()),
		CreatedAt:    deployment.CreatedAt().UTC().Format(time.RFC3339),
	}

	// Add optional fields
	if eventID, ok := deployment.TektonEventID(); ok {
		output.TektonEventID = eventID
	}

	if runName, ok := deployment.TektonPipelineRunName(); ok {
		output.TektonPipelineRunName = runName
	}

	if summary, ok := deployment.Summary(); ok {
		output.Summary = summary
	}

	if startedAt, ok := deployment.StartedAt(); ok {
		output.StartedAt = startedAt.UTC().Format(time.RFC3339)
	}

	if finishedAt, ok := deployment.FinishedAt(); ok {
		output.FinishedAt = finishedAt.UTC().Format(time.RFC3339)
	}

	uc.logger.Info(ctx, "get deployment completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint64("deployment_id", uint64(deployment.DeploymentID)),
		zap.String("status", string(deployment.Status())),
	)

	return output, nil
}
