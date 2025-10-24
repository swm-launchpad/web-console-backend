package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
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
}

func NewAddBuildVarUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
) *AddBuildVarUseCase {
	return &AddBuildVarUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
	}
}

func (uc *AddBuildVarUseCase) Execute(ctx context.Context, input AddBuildVarInput) (*AddBuildVarOutput, error) {
	var buildVarID uint
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

		// Add build variable
		buildVar, err := container.AddBuildVar(input.Key, input.Value)
		if err != nil {
			return err
		}

		// Mark container as needing rebuild since build variables changed
		container.MarkNeedsBuild()

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
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

	return &AddBuildVarOutput{
		ContainerID: input.ContainerID,
		BuildVarID:  buildVarID,
		Key:         key,
		Value:       value,
		CreatedAt:   createdAt,
	}, nil
}
