package application

import (
	"context"
	"strconv"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type DeleteProjectInput struct {
	ProjectID uint
}

type DeleteProjectOutput struct {
	Message string `json:"message"`
}

type DeleteProjectUseCase struct {
	projectService        service.ProjectService
	volumeService         service.VolumeService
	tektonCleanupClient   infrastructure.TektonCleanupClient
	containerSlugProvider infrastructure.ContainerSlugProvider
	txManager             db.TxManager
	logger                logger.Logger
}

func NewDeleteProjectUseCase(
	projectService service.ProjectService,
	volumeService service.VolumeService,
	tektonCleanupClient infrastructure.TektonCleanupClient,
	containerSlugProvider infrastructure.ContainerSlugProvider,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteProjectUseCase {
	return &DeleteProjectUseCase{
		projectService:        projectService,
		volumeService:         volumeService,
		tektonCleanupClient:   tektonCleanupClient,
		containerSlugProvider: containerSlugProvider,
		txManager:             txManager,
		logger:                log,
	}
}

func (uc *DeleteProjectUseCase) Execute(ctx context.Context, input DeleteProjectInput) (*DeleteProjectOutput, error) {
	uc.logger.Info(ctx, "delete project started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Variable to store project slug for Tekton cleanup
	var projectSlug string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// First, get the project to check operation status and get slug
		project, err := uc.projectService.GetProject(txCtx, input.ProjectID)
		if err != nil {
			uc.logger.Error(txCtx, "failed to get project for deletion",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Check if project has an operation in progress
		if project.OperationStatus() != value.ProjectOperationStatusNothing {
			uc.logger.Warn(txCtx, "cannot delete project: operation in progress",
				zap.Uint("project_id", input.ProjectID),
				zap.String("operation_status", string(project.OperationStatus())),
			)
			return projecterrors.ErrProjectOperationInProgress
		}

		// Store slug for later use in Tekton cleanup
		projectSlug = project.Slug().String()

		// Delete all volumes for the project (physical delete)
		if err := uc.volumeService.DeleteVolumesByProjectID(txCtx, input.ProjectID); err != nil {
			uc.logger.Error(txCtx, "failed to delete volumes for project",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Delete the project (soft delete)
		if err := uc.projectService.DeleteProject(txCtx, input.ProjectID); err != nil {
			uc.logger.Error(txCtx, "failed to delete project",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "delete project from database completed",
		zap.Uint("project_id", input.ProjectID),
		zap.String("project_slug", projectSlug),
	)

	// Get all container slugs (including soft-deleted) for cleanup
	containerSlugs, err := uc.containerSlugProvider.GetContainerSlugsByProjectID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Warn(ctx, "failed to get container slugs for cleanup (continuing with empty list)",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		// Continue with empty list - not fatal
		containerSlugs = []string{}
	}

	uc.logger.Info(ctx, "retrieved container slugs for cleanup",
		zap.Uint("project_id", input.ProjectID),
		zap.Int("container_count", len(containerSlugs)),
	)

	// Trigger Tekton cleanup pipeline (Tekton API returns 202 and runs asynchronously)
	// Pass project_id as string and container slugs (image names) for Tekton API
	projectIDStr := strconv.FormatUint(uint64(input.ProjectID), 10)
	if err := uc.tektonCleanupClient.TriggerCleanup(ctx, projectIDStr, "application", containerSlugs); err != nil {
		// Log the error but don't fail the deletion - Tekton cleanup is fire-and-forget
		uc.logger.Warn(ctx, "failed to trigger tekton cleanup pipeline (continuing with deletion)",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
			zap.String("project_slug", projectSlug),
			zap.Int("container_count", len(containerSlugs)),
		)
	} else {
		uc.logger.Info(ctx, "tekton cleanup pipeline triggered successfully",
			zap.Uint("project_id", input.ProjectID),
			zap.String("project_slug", projectSlug),
			zap.Int("container_count", len(containerSlugs)),
		)
	}

	uc.logger.Info(ctx, "delete project completed",
		zap.Uint("project_id", input.ProjectID),
	)

	return &DeleteProjectOutput{
		Message: "Project deleted successfully",
	}, nil
}
