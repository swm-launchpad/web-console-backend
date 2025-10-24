package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type DeleteSecretInput struct {
	ContainerID uint
	UserID      uint
	Key         string
}

type DeleteSecretOutput struct {
	ContainerID uint   `json:"container_id"`
	Key         string `json:"key"`
	DeletedAt   string `json:"deleted_at"`
}

type DeleteSecretUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewDeleteSecretUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteSecretUseCase {
	return &DeleteSecretUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *DeleteSecretUseCase) Execute(ctx context.Context, input DeleteSecretInput) (*DeleteSecretOutput, error) {
	uc.logger.Info(ctx, "delete secret started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("key", input.Key),
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

		// Delete secret
		if err := container.DeleteSecret(input.Key); err != nil {
			uc.logger.Error(ctx, "failed to delete secret",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.String("key", input.Key),
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

	uc.logger.Info(ctx, "delete secret completed",
		zap.Uint("container_id", input.ContainerID),
		zap.String("key", input.Key),
	)

	return &DeleteSecretOutput{
		ContainerID: input.ContainerID,
		Key:         input.Key,
		DeletedAt:   deletedAt,
	}, nil
}
