package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type UpdateBuildVarInput struct {
	ContainerID uint
	UserID      uint
	BuildVarKey string
	Value       string
}

type UpdateBuildVarOutput struct {
	ContainerID uint   `json:"container_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateBuildVarUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewUpdateBuildVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *UpdateBuildVarUseCase {
	return &UpdateBuildVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *UpdateBuildVarUseCase) Execute(ctx context.Context, input UpdateBuildVarInput) (*UpdateBuildVarOutput, error) {
	uc.logger.Info(ctx, "update build var started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("key", input.BuildVarKey),
	)

	var key, value, updatedAt string

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

		// Update build variable
		if err := container.UpdateBuildVar(input.BuildVarKey, input.Value); err != nil {
			uc.logger.Error(ctx, "failed to update build var",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.String("key", input.BuildVarKey),
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

		// Get updated build var
		buildVar, err := container.GetBuildVar(input.BuildVarKey)
		if err != nil {
			uc.logger.Error(ctx, "failed to get updated build var",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.String("key", input.BuildVarKey),
			)
			return err
		}

		// Extract values
		key = buildVar.Key()
		value = buildVar.Value()
		updatedAt = buildVar.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "update build var completed",
		zap.Uint("container_id", input.ContainerID),
		zap.String("key", key),
	)

	return &UpdateBuildVarOutput{
		ContainerID: input.ContainerID,
		Key:         key,
		Value:       value,
		UpdatedAt:   updatedAt,
	}, nil
}
