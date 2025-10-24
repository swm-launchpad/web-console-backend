package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type AddVolumeInput struct {
	ProjectID uint
	Name      string
	Capacity  uint32
}

type AddVolumeOutput struct {
	VolumeID  uint   `json:"volume_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Capacity  uint32 `json:"capacity"`
	CreatedAt string `json:"created_at"`
}

type AddVolumeUseCase struct {
	volumeService service.VolumeService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewAddVolumeUseCase(volumeService service.VolumeService, txManager db.TxManager, log logger.Logger) *AddVolumeUseCase {
	return &AddVolumeUseCase{
		volumeService: volumeService,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *AddVolumeUseCase) Execute(ctx context.Context, input AddVolumeInput) (*AddVolumeOutput, error) {
	uc.logger.Info(ctx, "add volume started",
		zap.Uint("project_id", input.ProjectID),
		zap.String("name", input.Name),
		zap.Uint32("capacity", input.Capacity),
	)

	var volumeID uint
	var projectID uint
	var name string
	var slug string
	var capacity uint32
	var createdAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Create volume through VolumeService
		volume, err := uc.volumeService.CreateVolume(txCtx, input.ProjectID, input.Name, input.Capacity)
		if err != nil {
			uc.logger.Error(ctx, "failed to create volume",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
				zap.String("name", input.Name),
			)
			return err
		}

		// Extract values within transaction
		volumeID = volume.VolumeID()
		projectID = volume.ProjectID()
		name = volume.Name()
		slug = volume.Slug().String()
		capacity = volume.Capacity()
		createdAt = volume.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "add volume completed",
		zap.Uint("volume_id", volumeID),
		zap.Uint("project_id", projectID),
		zap.String("slug", slug),
	)

	// Build output after successful transaction
	return &AddVolumeOutput{
		VolumeID:  volumeID,
		ProjectID: projectID,
		Name:      name,
		Slug:      slug,
		Capacity:  capacity,
		CreatedAt: createdAt,
	}, nil
}
