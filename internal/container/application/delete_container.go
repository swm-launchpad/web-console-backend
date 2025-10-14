package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type DeleteContainerInput struct {
	ContainerID uint
	UserID      uint
}

type DeleteContainerOutput struct {
	ContainerID uint   `json:"container_id"`
	DeletedAt   string `json:"deleted_at"`
}

type DeleteContainerUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
}

func NewDeleteContainerUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *DeleteContainerUseCase {
	return &DeleteContainerUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *DeleteContainerUseCase) Execute(ctx context.Context, input DeleteContainerInput) (*DeleteContainerOutput, error) {
	var containerID uint
	var deletedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			return err
		}

		// Get container with lock
		container, err := uc.containerRepo.FindByIDForUpdate(txCtx, input.ContainerID)
		if err != nil {
			return err
		}

		// Soft delete
		if err := container.SoftDelete(); err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		containerID = container.ContainerID()
		if container.DeletedAt() != nil {
			deletedAt = container.DeletedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DeleteContainerOutput{
		ContainerID: containerID,
		DeletedAt:   deletedAt,
	}, nil
}
