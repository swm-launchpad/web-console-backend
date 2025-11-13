package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckVolumeNameInput struct {
	ProjectID uint
	Name      string
}

type CheckVolumeNameOutput struct {
	Exists bool `json:"exists"`
}

type CheckVolumeNameUseCase struct {
	volumeRepo repository.VolumeRepository
	logger     logger.Logger
}

func NewCheckVolumeNameUseCase(volumeRepo repository.VolumeRepository, log logger.Logger) *CheckVolumeNameUseCase {
	return &CheckVolumeNameUseCase{
		volumeRepo: volumeRepo,
		logger:     log,
	}
}

func (uc *CheckVolumeNameUseCase) Execute(ctx context.Context, input CheckVolumeNameInput) (*CheckVolumeNameOutput, error) {
	uc.logger.Debug(ctx, "check volume name started",
		zap.Uint("project_id", input.ProjectID),
		zap.String("name", input.Name),
	)

	exists, err := uc.volumeRepo.ExistsByName(ctx, input.ProjectID, input.Name)
	if err != nil {
		uc.logger.Error(ctx, "failed to check volume name",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
			zap.String("name", input.Name),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check volume name completed",
		zap.Uint("project_id", input.ProjectID),
		zap.String("name", input.Name),
		zap.Bool("exists", exists),
	)

	return &CheckVolumeNameOutput{
		Exists: exists,
	}, nil
}
