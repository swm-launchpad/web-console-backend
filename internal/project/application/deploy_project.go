package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
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
}

func NewDeployProjectUseCase(
	deployService service.DeployService,
) *DeployProjectUseCase {
	return &DeployProjectUseCase{
		deployService: deployService,
	}
}

func (uc *DeployProjectUseCase) Execute(ctx context.Context, input DeployProjectInput) (*DeployProjectOutput, error) {
	// Note: Permission check is performed in the handler to prevent information disclosure
	// The handler converts permission errors to "not found" errors

	// Deploy project (userID is not needed as permission is already checked in handler)
	// We pass 0 as userID since it's not used by the service for actual deployment
	deployment, err := uc.deployService.DeployProject(ctx, input.ProjectID, 0)
	if err != nil {
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

	return output, nil
}
