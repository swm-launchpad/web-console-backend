package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
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
	projectService service.ProjectService
	volumeService  service.VolumeService
	txManager      db.TxManager
	logger         logger.Logger
}

func NewDeleteProjectUseCase(projectService service.ProjectService, volumeService service.VolumeService, txManager db.TxManager, log logger.Logger) *DeleteProjectUseCase {
	return &DeleteProjectUseCase{
		projectService: projectService,
		volumeService:  volumeService,
		txManager:      txManager,
		logger:         log,
	}
}

func (uc *DeleteProjectUseCase) Execute(ctx context.Context, input DeleteProjectInput) (*DeleteProjectOutput, error) {
	uc.logger.Info(ctx, "delete project started",
		zap.Uint("project_id", input.ProjectID),
	)

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// First, delete all volumes for the project (physical delete)
		if err := uc.volumeService.DeleteVolumesByProjectID(txCtx, input.ProjectID); err != nil {
			uc.logger.Error(ctx, "failed to delete volumes for project",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Then, delete the project (soft delete)
		if err := uc.projectService.DeleteProject(txCtx, input.ProjectID); err != nil {
			uc.logger.Error(ctx, "failed to delete project",
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

	uc.logger.Info(ctx, "delete project completed",
		zap.Uint("project_id", input.ProjectID),
	)

	return &DeleteProjectOutput{
		Message: "Project deleted successfully",
	}, nil
}
