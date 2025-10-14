package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type DeleteMountInput struct {
	ContainerID uint
	UserID      uint
	VolumeID    uint
}

type DeleteMountOutput struct {
	ContainerID uint   `json:"container_id"`
	VolumeID    uint   `json:"volume_id"`
	DeletedAt   string `json:"deleted_at"`
}

type DeleteMountUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
}

func NewDeleteMountUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *DeleteMountUseCase {
	return &DeleteMountUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *DeleteMountUseCase) Execute(ctx context.Context, input DeleteMountInput) (*DeleteMountOutput, error) {
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

		// Delete volume mount
		if err := container.DeleteMount(input.VolumeID); err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		deletedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DeleteMountOutput{
		ContainerID: input.ContainerID,
		VolumeID:    input.VolumeID,
		DeletedAt:   deletedAt,
	}, nil
}
