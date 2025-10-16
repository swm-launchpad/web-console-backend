package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type UpdateEnvVarInput struct {
	ContainerID uint
	UserID      uint
	EnvVarKey   string
	Value       string
}

type UpdateEnvVarOutput struct {
	ContainerID uint   `json:"container_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateEnvVarUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
}

func NewUpdateEnvVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *UpdateEnvVarUseCase {
	return &UpdateEnvVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *UpdateEnvVarUseCase) Execute(ctx context.Context, input UpdateEnvVarInput) (*UpdateEnvVarOutput, error) {
	var key, value, updatedAt string

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

		// Update environment variable
		if err := container.UpdateEnvVar(input.EnvVarKey, input.Value); err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Get updated env var
		envVar, err := container.GetEnvVar(input.EnvVarKey)
		if err != nil {
			return err
		}

		// Extract values
		key = envVar.Key()
		value = envVar.Value()
		updatedAt = envVar.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &UpdateEnvVarOutput{
		ContainerID: input.ContainerID,
		Key:         key,
		Value:       value,
		UpdatedAt:   updatedAt,
	}, nil
}
