package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
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
	logger        logger.Logger
}

func NewDeleteContainerUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteContainerUseCase {
	return &DeleteContainerUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *DeleteContainerUseCase) Execute(ctx context.Context, input DeleteContainerInput) (*DeleteContainerOutput, error) {
	uc.logger.Info(ctx, "delete container started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
	)

	var containerID uint
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

		// Soft delete
		if err := container.SoftDelete(); err != nil {
			uc.logger.Error(ctx, "failed to soft delete container",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
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
		containerID = container.ContainerID()
		if container.DeletedAt() != nil {
			deletedAt = container.DeletedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "delete container completed",
		zap.Uint("container_id", containerID),
	)

	return &DeleteContainerOutput{
		ContainerID: containerID,
		DeletedAt:   deletedAt,
	}, nil
}
