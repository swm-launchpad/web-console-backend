package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type DeployProjectInput struct {
	ProjectID uint
}

type DeployProjectOutput struct {
	DeploymentID          uint64 `json:"deployment_id"`
	ProjectID             uint   `json:"project_id"`
	Status                string `json:"status"`
	TektonEventID         string `json:"tekton_event_id,omitempty"`
	TektonPipelineRunName string `json:"tekton_pipeline_run_name,omitempty"`
	Summary               string `json:"summary,omitempty"`
	StartedAt             string `json:"started_at,omitempty"`
	CreatedAt             string `json:"created_at"`
}

type DeployProjectUseCase struct {
	deployService service.DeployService
	logger        logger.Logger
}

func NewDeployProjectUseCase(
	deployService service.DeployService,
	log logger.Logger,
) *DeployProjectUseCase {
	return &DeployProjectUseCase{
		deployService: deployService,
		logger:        log,
	}
}

func (uc *DeployProjectUseCase) Execute(ctx context.Context, input DeployProjectInput) (*DeployProjectOutput, error) {
	uc.logger.Info(ctx, "deploy project started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Note: Permission check is performed in the handler to prevent information disclosure
	// The handler converts permission errors to "not found" errors

	// Deploy project
	deployment, err := uc.deployService.DeployProject(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to deploy project",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	output := &DeployProjectOutput{
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

	uc.logger.Info(ctx, "deploy project completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint64("deployment_id", uint64(deployment.DeploymentID)),
		zap.String("status", string(deployment.Status())),
	)

	return output, nil
}
