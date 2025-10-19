package application

import (
	"context"
	"fmt"
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
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
	deployService  service.DeployService
	deploymentRepo repository.DeploymentRepository
}

func NewRefreshDeploymentUseCase(
	deployService service.DeployService,
	deploymentRepo repository.DeploymentRepository,
) *RefreshDeploymentUseCase {
	return &RefreshDeploymentUseCase{
		deployService:  deployService,
		deploymentRepo: deploymentRepo,
	}
}

func (uc *RefreshDeploymentUseCase) Execute(ctx context.Context, input RefreshDeploymentInput) (*RefreshDeploymentOutput, error) {
	// Note: Permission check is performed in the handler to prevent information disclosure
	// The handler converts permission errors to "not found" errors

	// Find active deployments for the project
	// According to the invariant analysis, there should be at most one active deployment per project
	activeDeployments, err := uc.deploymentRepo.FindActiveDeploymentsByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Check if there are any active deployments
	if len(activeDeployments) == 0 {
		// No active deployment - this is a valid state (project is not being deployed)
		return nil, projecterrors.ErrDeploymentNotFound
	}

	// Verify invariant: exactly one active deployment per project
	// This should never happen due to the deployment locking mechanism,
	// but we check defensively to catch potential bugs
	if len(activeDeployments) > 1 {
		// CRITICAL: Invariant violation detected
		// This indicates a serious bug in the deployment locking mechanism
		return nil, fmt.Errorf("invariant violation: project %d has %d active deployments (expected at most 1)",
			input.ProjectID, len(activeDeployments))
	}

	// Refresh the active deployment status
	deployment := activeDeployments[0]
	refreshedDeployment, err := uc.deployService.RefreshDeploymentStatus(ctx, uint64(deployment.DeploymentID))
	if err != nil {
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

	return output, nil
}
