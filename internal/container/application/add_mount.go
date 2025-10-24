package application

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
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
	containerRepo    repository.ContainerRepository
	permissionSvc    service.PermissionService
	projectVolumeSvc projectService.VolumeService
	txManager        db.TxManager
	logger           logger.Logger
}

func NewAddMountUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	projectVolumeSvc projectService.VolumeService,
	txManager db.TxManager,
	log logger.Logger,
) *AddMountUseCase {
	return &AddMountUseCase{
		containerRepo:    containerRepo,
		permissionSvc:    permissionSvc,
		projectVolumeSvc: projectVolumeSvc,
		txManager:        txManager,
		logger:           log,
	}
}

func (uc *AddMountUseCase) Execute(ctx context.Context, input AddMountInput) (*AddMountOutput, error) {
	uc.logger.Info(ctx, "add mount started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.Uint("volume_id", input.VolumeID),
		zap.String("mount_path", input.MountPath),
	)

	var volumeID uint
	var mountPath, createdAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			uc.logger.Warn(ctx, "permission check failed",
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

		// Validate volume exists and belongs to the same project
		volume, err := uc.projectVolumeSvc.GetVolume(txCtx, input.VolumeID)
		if err != nil {
			uc.logger.Error(ctx, "volume not found",
				zap.Error(err),
				zap.Uint("volume_id", input.VolumeID),
			)
			return errors.ErrVolumeNotFound
		}

		// Verify volume belongs to the same project as the container
		if volume.ProjectID() != container.ProjectID() {
			uc.logger.Error(ctx, "volume does not belong to project",
				zap.Uint("volume_id", input.VolumeID),
				zap.Uint("container_project_id", container.ProjectID()),
				zap.Uint("volume_project_id", volume.ProjectID()),
			)
			return fmt.Errorf("volume %d does not belong to project %d",
				input.VolumeID, container.ProjectID())
		}

		// Add volume mount
		mount, err := container.AddMount(input.VolumeID, input.MountPath)
		if err != nil {
			uc.logger.Error(ctx, "failed to add mount",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.Uint("volume_id", input.VolumeID),
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
		volumeID = mount.VolumeID()
		mountPath = mount.MountPath()
		createdAt = mount.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "add mount completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("volume_id", volumeID),
		zap.String("mount_path", mountPath),
	)

	return &AddMountOutput{
		ContainerID: input.ContainerID,
		VolumeID:    volumeID,
		MountPath:   mountPath,
		CreatedAt:   createdAt,
	}, nil
}
