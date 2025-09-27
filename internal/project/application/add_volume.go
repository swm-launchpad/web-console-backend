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
	var output *AddVolumeOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Create volume through VolumeService
		volume, err := uc.volumeService.CreateVolume(txCtx, input.ProjectID, input.Name, input.Capacity)
		if err != nil {
			return err
		}

		// Build output
		output = &AddVolumeOutput{
			VolumeID:  volume.GetVolumeID(),
			ProjectID: volume.GetProjectID(),
			Name:      volume.GetName(),
			Capacity:  volume.GetCapacity(),
			CreatedAt: volume.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		}

		return nil
	})

	return output, err
}
