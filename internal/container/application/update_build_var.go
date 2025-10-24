package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
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
}

func NewUpdateBuildVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *UpdateBuildVarUseCase {
	return &UpdateBuildVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *UpdateBuildVarUseCase) Execute(ctx context.Context, input UpdateBuildVarInput) (*UpdateBuildVarOutput, error) {
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

		// Update build variable
		if err := container.UpdateBuildVar(input.BuildVarKey, input.Value); err != nil {
			return err
		}

		// Mark container as needing rebuild since build variables changed
		container.MarkNeedsBuild()

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Get updated build var
		buildVar, err := container.GetBuildVar(input.BuildVarKey)
		if err != nil {
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

	return &UpdateBuildVarOutput{
		ContainerID: input.ContainerID,
		Key:         key,
		Value:       value,
		UpdatedAt:   updatedAt,
	}, nil
}
