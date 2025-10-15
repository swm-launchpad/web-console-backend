package application

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type AddMountInput struct {
	ContainerID uint
	UserID      uint
	VolumeID    uint
	MountPath   string
}

type AddMountOutput struct {
	ContainerID uint   `json:"container_id"`
	VolumeID    uint   `json:"volume_id"`
	MountPath   string `json:"mount_path"`
	CreatedAt   string `json:"created_at"`
}

type AddMountUseCase struct {
	containerRepo      repository.ContainerRepository
	permissionSvc      service.PermissionService
	projectVolumeSvc   projectService.VolumeService
	txManager          db.TxManager
}

func NewAddMountUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	projectVolumeSvc projectService.VolumeService,
	txManager db.TxManager,
) *AddMountUseCase {
	return &AddMountUseCase{
		containerRepo:    containerRepo,
		permissionSvc:    permissionSvc,
		projectVolumeSvc: projectVolumeSvc,
		txManager:        txManager,
	}
}

func (uc *AddMountUseCase) Execute(ctx context.Context, input AddMountInput) (*AddMountOutput, error) {
	var volumeID uint
	var mountPath, createdAt string

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

		// Validate volume exists and belongs to the same project
		volume, err := uc.projectVolumeSvc.GetVolume(txCtx, input.VolumeID)
		if err != nil {
			return errors.ErrVolumeNotFound
		}

		// Verify volume belongs to the same project as the container
		if volume.ProjectID() != container.ProjectID() {
			return fmt.Errorf("%w: volume %d does not belong to project %d",
				errors.ErrInvalidInput, input.VolumeID, container.ProjectID())
		}

		// Add volume mount
		mount, err := container.AddMount(input.VolumeID, input.MountPath)
		if err != nil {
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			return err
		}

		// Extract values
		volumeID = mount.VolumeID()
		mountPath = mount.MountPath()
		createdAt = mount.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &AddMountOutput{
		ContainerID: input.ContainerID,
		VolumeID:    volumeID,
		MountPath:   mountPath,
		CreatedAt:   createdAt,
	}, nil
}
