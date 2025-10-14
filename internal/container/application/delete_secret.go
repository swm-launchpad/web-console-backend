package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
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
}

func NewDeleteSecretUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *DeleteSecretUseCase {
	return &DeleteSecretUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *DeleteSecretUseCase) Execute(ctx context.Context, input DeleteSecretInput) (*DeleteSecretOutput, error) {
	var deletedAt string

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

		// Delete secret
		if err := container.DeleteSecret(input.Key); err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		deletedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DeleteSecretOutput{
		ContainerID: input.ContainerID,
		Key:         input.Key,
		DeletedAt:   deletedAt,
	}, nil
}
