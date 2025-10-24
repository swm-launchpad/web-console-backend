package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type GetVolumesInput struct {
	ProjectID uint
}

type VolumeListItem struct {
	VolumeID  uint   `json:"volume_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Capacity  uint32 `json:"capacity"`
	CreatedAt string `json:"created_at"`
}

type GetVolumesOutput struct {
	Volumes []VolumeListItem `json:"volumes"`
}

type GetVolumesUseCase struct {
	volumeService service.VolumeService
	logger        logger.Logger
}

func NewGetVolumesUseCase(volumeService service.VolumeService, log logger.Logger) *GetVolumesUseCase {
	return &GetVolumesUseCase{
		volumeService: volumeService,
		logger:        log,
	}
}

func (uc *GetVolumesUseCase) Execute(ctx context.Context, input GetVolumesInput) (*GetVolumesOutput, error) {
	uc.logger.Info(ctx, "get volumes started",
		zap.Uint("project_id", input.ProjectID),
	)

	// Get volumes for specific project
	volumes, err := uc.volumeService.ListVolumesByProjectID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get volumes",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
		)
		return nil, err
	}

	// Build output
	output := &GetVolumesOutput{
		Volumes: make([]VolumeListItem, 0, len(volumes)),
	}

	for _, volume := range volumes {
		output.Volumes = append(output.Volumes, VolumeListItem{
			VolumeID:  volume.VolumeID(),
			ProjectID: volume.ProjectID(),
			Name:      volume.Name(),
			Slug:      volume.Slug().String(),
			Capacity:  volume.Capacity(),
			CreatedAt: volume.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	uc.logger.Info(ctx, "get volumes completed",
		zap.Uint("project_id", input.ProjectID),
		zap.Int("count", len(volumes)),
	)

	return output, nil
}
