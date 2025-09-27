package application

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type GetProjectInput struct {
	ProjectID *uint
	Slug      *string
}

type ProjectUserOutput struct {
	UserID    uint   `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type VolumeOutput struct {
	VolumeID  uint   `json:"volume_id"`
	Name      string `json:"name"`
	Capacity  uint32 `json:"capacity"`
	CreatedAt string `json:"created_at"`
}

type GetProjectOutput struct {
	ProjectID    uint                `json:"project_id"`
	Name         string              `json:"name"`
	Slug         string              `json:"slug"`
	FQDN         string              `json:"fqdn,omitempty"`
	Plan         string              `json:"plan,omitempty"`
	Status       string              `json:"status"`
	CPULimit     *uint32             `json:"cpu_limit,omitempty"`
	MemoryLimit  *uint32             `json:"memory_limit,omitempty"`
	DiskLimit    *uint32             `json:"disk_limit,omitempty"`
	TrafficLimit *uint64             `json:"traffic_limit,omitempty"`
	Users        []ProjectUserOutput `json:"users"`
	Volumes      []VolumeOutput      `json:"volumes"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at,omitempty"`
}

type GetProjectUseCase struct {
	projectService service.ProjectService
	volumeService  service.VolumeService
}

func NewGetProjectUseCase(projectService service.ProjectService, volumeService service.VolumeService) *GetProjectUseCase {
	return &GetProjectUseCase{
		projectService: projectService,
		volumeService:  volumeService,
	}
}

func (uc *GetProjectUseCase) Execute(ctx context.Context, input GetProjectInput) (*GetProjectOutput, error) {
	var project *model.Project
	var err error

	// Get project by ID or slug
	if input.ProjectID != nil {
		project, err = uc.projectService.GetProject(ctx, *input.ProjectID)
	} else if input.Slug != nil {
		project, err = uc.projectService.GetProjectBySlug(ctx, *input.Slug)
	} else {
		return nil, projecterrors.ErrInvalidProjectID
	}

	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetProjectOutput{
		ProjectID:    project.GetProjectID(),
		Name:         project.GetName(),
		Slug:         project.GetSlug().String(),
		Status:       string(project.GetStatus()),
		CPULimit:     project.GetLimits().GetCPULimit(),
		MemoryLimit:  project.GetLimits().GetMemoryLimit(),
		DiskLimit:    project.GetLimits().GetDiskLimit(),
		TrafficLimit: project.GetLimits().GetTrafficLimit(),
		CreatedAt:    project.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		Users:        make([]ProjectUserOutput, 0),
		Volumes:      make([]VolumeOutput, 0),
	}

	if project.GetFQDN() != nil {
		output.FQDN = *project.GetFQDN()
	}

	if project.GetPlan() != nil {
		output.Plan = *project.GetPlan()
	}

	if project.GetUpdatedAt() != nil {
		output.UpdatedAt = project.GetUpdatedAt().Format("2006-01-02T15:04:05Z")
	}

	// Add users
	for _, user := range project.GetActiveUsers() {
		output.Users = append(output.Users, ProjectUserOutput{
			UserID:    user.GetUserID(),
			Role:      string(user.GetRole()),
			CreatedAt: user.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	// Add volumes from VolumeService
	volumes, err := uc.volumeService.GetVolumesByProjectID(ctx, project.GetProjectID())
	if err == nil {
		for _, volume := range volumes {
			output.Volumes = append(output.Volumes, VolumeOutput{
				VolumeID:  volume.GetVolumeID(),
				Name:      volume.GetName(),
				Capacity:  volume.GetCapacity(),
				CreatedAt: volume.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			})
		}
	}

	return output, nil
}
