package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type UpdateVolumeInput struct {
	VolumeID uint
	Name     *string
	Capacity *uint32
}

type UpdateVolumeOutput struct {
	VolumeID  uint   `json:"volume_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Capacity  uint32 `json:"capacity"`
	UpdatedAt string `json:"updated_at"`
}

type UpdateVolumeUseCase struct {
	volumeService service.VolumeService
	txManager     db.TxManager
}

func NewUpdateVolumeUseCase(
	volumeService service.VolumeService,
	txManager db.TxManager,
) *UpdateVolumeUseCase {
	return &UpdateVolumeUseCase{
		volumeService: volumeService,
		txManager:     txManager,
	}
}

func (uc *UpdateVolumeUseCase) Execute(ctx context.Context, input UpdateVolumeInput) (*UpdateVolumeOutput, error) {
	var output *UpdateVolumeOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Get current volume to check existing values
		currentVolume, err := uc.volumeService.GetVolume(txCtx, input.VolumeID)
		if err != nil {
			return err
		}

		// Prepare update parameters
		name := currentVolume.GetName()
		if input.Name != nil {
			name = *input.Name
		}

		capacity := currentVolume.GetCapacity()
		if input.Capacity != nil {
			capacity = *input.Capacity
		}

		// Update volume through VolumeService
		volume, err := uc.volumeService.UpdateVolume(txCtx, input.VolumeID, name, capacity)
		if err != nil {
			return err
		}

		output = &UpdateVolumeOutput{
			VolumeID:  volume.GetVolumeID(),
			ProjectID: volume.GetProjectID(),
			Name:      volume.GetName(),
			Capacity:  volume.GetCapacity(),
		}
		if volume.GetUpdatedAt() != nil {
			output.UpdatedAt = volume.GetUpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}
