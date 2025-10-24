package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type UpdateSecretInput struct {
	ContainerID uint
	UserID      uint
	Key         string
	Value       string
}

type UpdateSecretOutput struct {
	ContainerID uint   `json:"container_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateSecretUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewUpdateSecretUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *UpdateSecretUseCase {
	return &UpdateSecretUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *UpdateSecretUseCase) Execute(ctx context.Context, input UpdateSecretInput) (*UpdateSecretOutput, error) {
	uc.logger.Info(ctx, "update secret started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("key", input.Key),
	)

	var value, updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			uc.logger.Warn(ctx, "permission check failed",
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

		// Update secret
		if err := container.UpdateSecret(input.Key, input.Value); err != nil {
			uc.logger.Error(ctx, "failed to update secret",
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
		secret, _ := container.GetSecret(input.Key)
		value = secret.Value()
		updatedAt = secret.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "update secret completed",
		zap.Uint("container_id", input.ContainerID),
		zap.String("key", input.Key),
	)

	return &UpdateSecretOutput{
		ContainerID: input.ContainerID,
		Key:         input.Key,
		Value:       value,
		UpdatedAt:   updatedAt,
	}, nil
}
