package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
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
}

func NewAddEnvVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *AddEnvVarUseCase {
	return &AddEnvVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *AddEnvVarUseCase) Execute(ctx context.Context, input AddEnvVarInput) (*AddEnvVarOutput, error) {
	var envVarID uint
	var key, value, createdAt string

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

		// Add environment variable
		envVar, err := container.AddEnvVar(input.Key, input.Value)
		if err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
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

	return &AddEnvVarOutput{
		ContainerID: input.ContainerID,
		EnvVarID:    envVarID,
		Key:         key,
		Value:       value,
		CreatedAt:   createdAt,
	}, nil
}
