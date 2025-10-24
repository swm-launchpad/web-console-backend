package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type DeleteBuildVarInput struct {
	ContainerID uint
	UserID      uint
	Key         string
}

type DeleteBuildVarOutput struct {
	ContainerID uint   `json:"container_id"`
	Key         string `json:"key"`
	DeletedAt   string `json:"deleted_at"`
}

type DeleteBuildVarUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewDeleteBuildVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteBuildVarUseCase {
	return &DeleteBuildVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *DeleteBuildVarUseCase) Execute(ctx context.Context, input DeleteBuildVarInput) (*DeleteBuildVarOutput, error) {
	uc.logger.Info(ctx, "delete build var started",
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

		// Delete build variable
		if err := container.DeleteBuildVar(input.Key); err != nil {
			uc.logger.Error(ctx, "failed to delete build var",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.String("key", input.Key),
			)
			return err
		}

		// Mark container as needing rebuild since build variables changed
		container.MarkNeedsBuild()

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

	uc.logger.Info(ctx, "delete build var completed",
		zap.Uint("container_id", input.ContainerID),
		zap.String("key", input.Key),
	)

	return &DeleteBuildVarOutput{
		ContainerID: input.ContainerID,
		Key:         input.Key,
		DeletedAt:   deletedAt,
	}, nil
}
