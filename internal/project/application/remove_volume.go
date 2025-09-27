package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type RemoveVolumeInput struct {
	VolumeID uint
}

type RemoveVolumeOutput struct {
	Message string `json:"message"`
}

type RemoveVolumeUseCase struct {
	volumeService service.VolumeService
	txManager     db.TxManager
}

func NewRemoveVolumeUseCase(
	volumeService service.VolumeService,
	txManager db.TxManager,
) *RemoveVolumeUseCase {
	return &RemoveVolumeUseCase{
		volumeService: volumeService,
		txManager:     txManager,
	}
}

func (uc *RemoveVolumeUseCase) Execute(ctx context.Context, input RemoveVolumeInput) (*RemoveVolumeOutput, error) {
	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete volume through VolumeService
		return uc.volumeService.DeleteVolume(txCtx, input.VolumeID)
	})

	if err != nil {
		return nil, err
	}

	return &RemoveVolumeOutput{
		Message: "Volume removed successfully",
	}, nil
}
