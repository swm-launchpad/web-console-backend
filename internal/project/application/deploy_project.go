package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/deploy"
	"go.uber.org/zap"
)

type DeployProjectInput struct {
	ProjectID uint
}

type DeployProjectOutput struct {
	Message   string `json:"message"`
	ProjectID uint   `json:"project_id"`
}

type DeployProjectUseCase struct {
	deployService deploy.Deployer
	logger        logger.Logger
}

func NewDeployProjectUseCase(
	deployService deploy.Deployer,
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

	// Build and deploy project
	// This initiates builds for all containers and returns immediately (202 Accepted)
	// Actual builds run in background goroutines
	err := uc.deployService.BuildAndDeployProject(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to initiate build and deploy",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	output := &DeployProjectOutput{
		Message:   "Build and deployment initiated",
		ProjectID: input.ProjectID,
	}

	uc.logger.Info(ctx, "deploy project initiated",
		zap.Uint("project_id", input.ProjectID),
	)

	return output, nil
}
