package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
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
}

func NewUpdateSecretUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *UpdateSecretUseCase {
	return &UpdateSecretUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *UpdateSecretUseCase) Execute(ctx context.Context, input UpdateSecretInput) (*UpdateSecretOutput, error) {
	var value, updatedAt string

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

		// Update secret
		if err := container.UpdateSecret(input.Key, input.Value); err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
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

	return &UpdateSecretOutput{
		ContainerID: input.ContainerID,
		Key:         input.Key,
		Value:       value,
		UpdatedAt:   updatedAt,
	}, nil
}
