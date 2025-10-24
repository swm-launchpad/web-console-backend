package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
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
	logger        logger.Logger
}

func NewDeleteNetworkUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *DeleteNetworkUseCase {
	return &DeleteNetworkUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *DeleteNetworkUseCase) Execute(ctx context.Context, input DeleteNetworkInput) (*DeleteNetworkOutput, error) {
	uc.logger.Info(ctx, "delete network started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.Uint16("internal_port", input.InternalPort),
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

		// Delete network by internal port
		if err := container.DeleteNetworkByInternalPort(input.InternalPort); err != nil {
			uc.logger.Error(ctx, "failed to delete network",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.Uint16("internal_port", input.InternalPort),
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
		deletedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "delete network completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint16("internal_port", input.InternalPort),
	)

	return &DeleteNetworkOutput{
		ContainerID:  input.ContainerID,
		InternalPort: input.InternalPort,
		DeletedAt:    deletedAt,
	}, nil
}
