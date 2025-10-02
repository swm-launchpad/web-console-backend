package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
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
	Capacity  uint32 `json:"capacity"`
	CreatedAt string `json:"created_at"`
}

type AddVolumeUseCase struct {
	volumeService service.VolumeService
	txManager     db.TxManager
}

func NewAddVolumeUseCase(volumeService service.VolumeService, txManager db.TxManager) *AddVolumeUseCase {
	return &AddVolumeUseCase{
		volumeService: volumeService,
		txManager:     txManager,
	}
}

func (uc *AddVolumeUseCase) Execute(ctx context.Context, input AddVolumeInput) (*AddVolumeOutput, error) {
	var volumeID uint
	var projectID uint
	var name string
	var capacity uint32
	var createdAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Create volume through VolumeService
		volume, err := uc.volumeService.CreateVolume(txCtx, input.ProjectID, input.Name, input.Capacity)
		if err != nil {
			return err
		}

		// Extract values within transaction
		volumeID = volume.VolumeID()
		projectID = volume.ProjectID()
		name = volume.Name()
		capacity = volume.Capacity()
		createdAt = volume.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build output after successful transaction
	return &AddVolumeOutput{
		VolumeID:  volumeID,
		ProjectID: projectID,
		Name:      name,
		Capacity:  capacity,
		CreatedAt: createdAt,
	}, nil
}
