package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
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
	logger        logger.Logger
}

func NewDeleteMountUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteMountUseCase {
	return &DeleteMountUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *DeleteMountUseCase) Execute(ctx context.Context, input DeleteMountInput) (*DeleteMountOutput, error) {
	uc.logger.Info(ctx, "delete mount started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.Uint("volume_id", input.VolumeID),
	)

	var deletedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			uc.logger.Error(ctx, "permission check failed",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Get container with lock
		container, err := uc.containerRepo.FindByIDForUpdate(txCtx, input.ContainerID)
		if err != nil {
			uc.logger.Error(ctx, "failed to find container for update",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Delete volume mount
		if err := container.DeleteMount(input.VolumeID); err != nil {
			uc.logger.Error(ctx, "failed to delete mount",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.Uint("volume_id", input.VolumeID),
			)
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			uc.logger.Error(ctx, "failed to save container",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Extract values
		deletedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "delete mount completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("volume_id", input.VolumeID),
	)

	return &DeleteMountOutput{
		ContainerID: input.ContainerID,
		VolumeID:    input.VolumeID,
		DeletedAt:   deletedAt,
	}, nil
}
