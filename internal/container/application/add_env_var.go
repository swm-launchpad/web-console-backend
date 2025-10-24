package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type AddEnvVarInput struct {
	ContainerID uint
	UserID      uint
	Key         string
	Value       string
}

type AddEnvVarOutput struct {
	ContainerID uint   `json:"container_id"`
	EnvVarID    uint   `json:"env_var_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	CreatedAt   string `json:"created_at"`
}

type AddEnvVarUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewAddEnvVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *AddEnvVarUseCase {
	return &AddEnvVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *AddEnvVarUseCase) Execute(ctx context.Context, input AddEnvVarInput) (*AddEnvVarOutput, error) {
	uc.logger.Info(ctx, "add env var started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("key", input.Key),
	)

	var envVarID uint
	var key, value, createdAt string

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

		// Add environment variable
		envVar, err := container.AddEnvVar(input.Key, input.Value)
		if err != nil {
			uc.logger.Error(ctx, "failed to add env var",
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
		envVarID = envVar.EnvVarID()
		key = envVar.Key()
		value = envVar.Value()
		createdAt = envVar.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "add env var completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("env_var_id", envVarID),
		zap.String("key", key),
	)

	return &AddEnvVarOutput{
		ContainerID: input.ContainerID,
		EnvVarID:    envVarID,
		Key:         key,
		Value:       value,
		CreatedAt:   createdAt,
	}, nil
}
