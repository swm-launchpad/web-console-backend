package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
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
	logger        logger.Logger
}

func NewRemoveVolumeUseCase(
	volumeService service.VolumeService,
	txManager db.TxManager,
	log logger.Logger,
) *RemoveVolumeUseCase {
	return &RemoveVolumeUseCase{
		volumeService: volumeService,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *RemoveVolumeUseCase) Execute(ctx context.Context, input RemoveVolumeInput) (*RemoveVolumeOutput, error) {
	uc.logger.Info(ctx, "remove volume started",
		zap.Uint("volume_id", input.VolumeID),
	)

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete volume through VolumeService
		if err := uc.volumeService.DeleteVolume(txCtx, input.VolumeID); err != nil {
			uc.logger.Error(ctx, "failed to delete volume",
				zap.Error(err),
				zap.Uint("volume_id", input.VolumeID),
			)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "remove volume completed",
		zap.Uint("volume_id", input.VolumeID),
	)

	return &RemoveVolumeOutput{
		Message: "Volume removed successfully",
	}, nil
}
