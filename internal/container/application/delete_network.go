package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type DeleteNetworkInput struct {
	ContainerID  uint
	UserID       uint
	InternalPort uint16
}

type DeleteNetworkOutput struct {
	ContainerID  uint   `json:"container_id"`
	InternalPort uint16 `json:"internal_port"`
	DeletedAt    string `json:"deleted_at"`
}

type DeleteNetworkUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
}

func NewDeleteNetworkUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *DeleteNetworkUseCase {
	return &DeleteNetworkUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *DeleteNetworkUseCase) Execute(ctx context.Context, input DeleteNetworkInput) (*DeleteNetworkOutput, error) {
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

		// Delete network by internal port
		if err := container.DeleteNetworkByInternalPort(input.InternalPort); err != nil {
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

	return &DeleteNetworkOutput{
		ContainerID:  input.ContainerID,
		InternalPort: input.InternalPort,
		DeletedAt:    deletedAt,
	}, nil
}
