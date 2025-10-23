package application

import (
	"context"

	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type GetProjectInput struct {
	ProjectID uint
}

type ProjectUserOutput struct {
	UserID    uint   `json:"user_id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type VolumeOutput struct {
	VolumeID  uint   `json:"volume_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
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
	CPULimit     uint32              `json:"cpu_limit"`
	MemoryLimit  uint32              `json:"memory_limit"`
	DiskLimit    uint32              `json:"disk_limit"`
	TrafficLimit uint32              `json:"traffic_limit"`
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
	// Get project by ID
	project, err := uc.projectService.GetProject(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	return buildProjectOutput(ctx, project, uc.volumeService)
}

// buildProjectOutput is a shared helper function to build GetProjectOutput from a project domain model
// This eliminates code duplication between GetProjectUseCase and GetProjectBySlugUseCase
func buildProjectOutput(ctx context.Context, project *model.Project, volumeService service.VolumeService) (*GetProjectOutput, error) {
	// Build output
	output := &GetProjectOutput{
		ProjectID:    project.ProjectID(),
		Name:         project.Name(),
		Slug:         project.Slug().String(),
		Status:       string(project.Status()),
		CPULimit:     project.Limits().CPULimit(),
		MemoryLimit:  project.Limits().MemoryLimit(),
		DiskLimit:    project.Limits().DiskLimit(),
		TrafficLimit: project.Limits().TrafficLimit(),
		CreatedAt:    project.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    project.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		Users:        make([]ProjectUserOutput, 0),
		Volumes:      make([]VolumeOutput, 0),
	}

	if fqdn, ok := project.FQDN(); ok {
		output.FQDN = fqdn
	}

	if plan, ok := project.Plan(); ok {
		output.Plan = plan
	}

	// Add users
	for _, user := range project.GetActiveUsers() {
		output.Users = append(output.Users, ProjectUserOutput{
			UserID:    user.UserID(),
			Role:      string(user.Role()),
			CreatedAt: user.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	// Add volumes from VolumeService
	volumes, err := volumeService.ListVolumesByProjectID(ctx, project.ProjectID())
	if err != nil {
		return nil, err
	}

	for _, volume := range volumes {
		output.Volumes = append(output.Volumes, VolumeOutput{
			VolumeID:  volume.VolumeID(),
			Name:      volume.Name(),
			Slug:      volume.Slug().String(),
			Capacity:  volume.Capacity(),
			CreatedAt: volume.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	return output, nil
}
