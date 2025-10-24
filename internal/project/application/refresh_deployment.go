package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type RefreshDeploymentInput struct {
	ProjectID uint
}

type RefreshDeploymentOutput struct {
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

type RefreshDeploymentUseCase struct {
	deployService service.DeployService
	logger        logger.Logger
}

func NewRefreshDeploymentUseCase(
	deployService service.DeployService,
	log logger.Logger,
) *RefreshDeploymentUseCase {
	return &RefreshDeploymentUseCase{
		deployService: deployService,
		logger:        log,
	}
}

func (uc *RefreshDeploymentUseCase) Execute(ctx context.Context, input RefreshDeploymentInput) (*RefreshDeploymentOutput, error) {
	uc.logger.Info(ctx, "refresh deployment started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Note: Permission check is performed in the handler to prevent information disclosure
	// The handler converts permission errors to "not found" errors

	// Refresh the active deployment for the project
	// This uses project.active_deployment_id internally
	refreshedDeployment, err := uc.deployService.RefreshActiveDeployment(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to refresh deployment",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	output := &RefreshDeploymentOutput{
		DeploymentID: uint64(refreshedDeployment.DeploymentID),
		ProjectID:    uint(refreshedDeployment.ProjectID()),
		Status:       string(refreshedDeployment.Status()),
		CreatedAt:    refreshedDeployment.CreatedAt().UTC().Format(time.RFC3339),
	}

	// Add optional fields
	if eventID, ok := refreshedDeployment.TektonEventID(); ok {
		output.TektonEventID = eventID
	}

	if runName, ok := refreshedDeployment.TektonPipelineRunName(); ok {
		output.TektonPipelineRunName = runName
	}

	if summary, ok := refreshedDeployment.Summary(); ok {
		output.Summary = summary
	}

	if startedAt, ok := refreshedDeployment.StartedAt(); ok {
		output.StartedAt = startedAt.UTC().Format(time.RFC3339)
	}

	if finishedAt, ok := refreshedDeployment.FinishedAt(); ok {
		output.FinishedAt = finishedAt.UTC().Format(time.RFC3339)
	}

	uc.logger.Info(ctx, "refresh deployment completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint64("deployment_id", uint64(refreshedDeployment.DeploymentID)),
		zap.String("status", string(refreshedDeployment.Status())),
	)

	return output, nil
}
