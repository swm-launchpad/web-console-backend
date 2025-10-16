package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type AddSecretInput struct {
	ContainerID uint
	UserID      uint
	Key         string
	Value       string
}

type AddSecretOutput struct {
	ContainerID uint   `json:"container_id"`
	SecretID    uint   `json:"secret_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	CreatedAt   string `json:"created_at"`
}

type AddSecretUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
}

func NewAddSecretUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *AddSecretUseCase {
	return &AddSecretUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *AddSecretUseCase) Execute(ctx context.Context, input AddSecretInput) (*AddSecretOutput, error) {
	var secretID uint
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

		// Add secret
		secret, err := container.AddSecret(input.Key, input.Value)
		if err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		secretID = secret.SecretID()
		key = secret.Key()
		value = secret.Value()
		createdAt = secret.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &AddSecretOutput{
		ContainerID: input.ContainerID,
		SecretID:    secretID,
		Key:         key,
		Value:       value,
		CreatedAt:   createdAt,
	}, nil
}
