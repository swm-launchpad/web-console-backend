package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type AddBuildVarInput struct {
	ContainerID uint
	UserID      uint
	Key         string
	Value       string
}

type AddBuildVarOutput struct {
	ContainerID uint   `json:"container_id"`
	BuildVarID  uint   `json:"build_var_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	CreatedAt   string `json:"created_at"`
}

type AddBuildVarUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewAddBuildVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *AddBuildVarUseCase {
	return &AddBuildVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *AddBuildVarUseCase) Execute(ctx context.Context, input AddBuildVarInput) (*AddBuildVarOutput, error) {
	uc.logger.Info(ctx, "add build var started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("key", input.Key),
	)

	var buildVarID uint
	var key, value, createdAt string

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

		// Add build variable
		buildVar, err := container.AddBuildVar(input.Key, input.Value)
		if err != nil {
			uc.logger.Error(ctx, "failed to add build var",
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
		buildVarID = buildVar.BuildVarID()
		key = buildVar.Key()
		value = buildVar.Value()
		createdAt = buildVar.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "add build var completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("build_var_id", buildVarID),
		zap.String("key", key),
	)

	return &AddBuildVarOutput{
		ContainerID: input.ContainerID,
		BuildVarID:  buildVarID,
		Key:         key,
		Value:       value,
		CreatedAt:   createdAt,
	}, nil
}
