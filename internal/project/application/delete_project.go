package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
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
}

func NewDeleteProjectUseCase(projectService service.ProjectService, volumeService service.VolumeService, txManager db.TxManager) *DeleteProjectUseCase {
	return &DeleteProjectUseCase{
		projectService: projectService,
		volumeService:  volumeService,
		txManager:      txManager,
	}
}

func (uc *DeleteProjectUseCase) Execute(ctx context.Context, input DeleteProjectInput) (*DeleteProjectOutput, error) {
	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// First, delete all volumes for the project (physical delete)
		if err := uc.volumeService.DeleteVolumesByProjectID(txCtx, input.ProjectID); err != nil {
			return err
		}

		// Then, delete the project (soft delete)
		return uc.projectService.DeleteProject(txCtx, input.ProjectID)
	})

	if err != nil {
		return nil, err
	}

	return &DeleteProjectOutput{
		Message: "Project deleted successfully",
	}, nil
}
