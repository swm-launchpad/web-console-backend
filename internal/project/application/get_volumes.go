package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type GetVolumesInput struct {
	ProjectID *uint
}

type VolumeListItem struct {
	VolumeID  uint   `json:"volume_id"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Capacity  uint32 `json:"capacity"`
	CreatedAt string `json:"created_at"`
}

type GetVolumesOutput struct {
	Volumes []VolumeListItem `json:"volumes"`
}

type GetVolumesUseCase struct {
	volumeService service.VolumeService
}

func NewGetVolumesUseCase(volumeService service.VolumeService) *GetVolumesUseCase {
	return &GetVolumesUseCase{
		volumeService: volumeService,
	}
}

func (uc *GetVolumesUseCase) Execute(ctx context.Context, input GetVolumesInput) (*GetVolumesOutput, error) {
	if input.ProjectID == nil {
		// Get all volumes across all projects
		volumes, err := uc.volumeService.ListVolumes(ctx, 0, 100) // Default pagination
		if err != nil {
			return nil, err
		}

		// Build output
		output := &GetVolumesOutput{
			Volumes: make([]VolumeListItem, 0, len(volumes)),
		}

		for _, volume := range volumes {
			output.Volumes = append(output.Volumes, VolumeListItem{
				VolumeID:  volume.GetVolumeID(),
				ProjectID: volume.GetProjectID(),
				Name:      volume.GetName(),
				Capacity:  volume.GetCapacity(),
				CreatedAt: volume.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			})
		}

		return output, nil
	}

	// Get volumes for specific project
	volumes, err := uc.volumeService.GetVolumesByProjectID(ctx, *input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetVolumesOutput{
		Volumes: make([]VolumeListItem, 0, len(volumes)),
	}

	for _, volume := range volumes {
		output.Volumes = append(output.Volumes, VolumeListItem{
			VolumeID:  volume.GetVolumeID(),
			ProjectID: volume.GetProjectID(),
			Name:      volume.GetName(),
			Capacity:  volume.GetCapacity(),
			CreatedAt: volume.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	return output, nil
}
