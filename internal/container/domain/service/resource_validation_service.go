package service

import (
	"context"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	projectrepository "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// ResourceValidationService validates container resource limits against project limits
type ResourceValidationService interface {
	// ValidateProjectResourceLimits checks if adding/updating a container would exceed project limits
	// excludeContainerID is used when updating an existing container (pass 0 for new containers)
	ValidateProjectResourceLimits(ctx context.Context, projectID uint, requestedCPU, requestedMemory uint32, excludeContainerID uint) error
}

type resourceValidationService struct {
	containerRepo repository.ContainerRepository
	projectRepo   projectrepository.ProjectRepository
}

// NewResourceValidationService creates a new resource validation service
func NewResourceValidationService(
	containerRepo repository.ContainerRepository,
	projectRepo projectrepository.ProjectRepository,
) ResourceValidationService {
	return &resourceValidationService{
		containerRepo: containerRepo,
		projectRepo:   projectRepo,
	}
}

// ValidateProjectResourceLimits checks if adding/updating a container would exceed project limits
func (s *resourceValidationService) ValidateProjectResourceLimits(
	ctx context.Context,
	projectID uint,
	requestedCPU, requestedMemory uint32,
	excludeContainerID uint,
) error {
	// Get project to retrieve resource limits
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return containererrors.ErrProjectNotFound
	}

	// Get total resource usage by project
	totalCPU, totalMemory, err := s.containerRepo.GetTotalResourceUsageByProject(ctx, projectID)
	if err != nil {
		return err
	}

	// If updating an existing container, subtract its current usage
	if excludeContainerID != 0 {
		container, err := s.containerRepo.FindByID(ctx, excludeContainerID)
		if err != nil {
			return err
		}

		if cpuLimit := container.ResourceLimits().CPULimit(); cpuLimit != nil {
			totalCPU -= *cpuLimit
		}
		if memoryLimit := container.ResourceLimits().MemoryLimit(); memoryLimit != nil {
			totalMemory -= *memoryLimit
		}
	}

	// Calculate new total with requested resources
	newTotalCPU := totalCPU + requestedCPU
	newTotalMemory := totalMemory + requestedMemory

	// Validate against project limits
	projectLimits := project.Limits()

	if newTotalCPU > projectLimits.CPULimit() {
		return containererrors.ErrProjectCPULimitExceeded
	}

	if newTotalMemory > projectLimits.MemoryLimit() {
		return containererrors.ErrProjectMemoryLimitExceeded
	}

	return nil
}
