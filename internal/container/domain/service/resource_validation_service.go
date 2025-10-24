package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
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
	logger        logger.Logger
}

// NewResourceValidationService creates a new resource validation service
func NewResourceValidationService(
	containerRepo repository.ContainerRepository,
	projectRepo projectrepository.ProjectRepository,
	log logger.Logger,
) ResourceValidationService {
	return &resourceValidationService{
		containerRepo: containerRepo,
		projectRepo:   projectRepo,
		logger:        log,
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
		s.logger.Warn(ctx, "Project CPU limit exceeded",
			zap.Uint("project_id", projectID),
			zap.Uint32("current_cpu", totalCPU),
			zap.Uint32("requested_cpu", requestedCPU),
			zap.Uint32("new_total_cpu", newTotalCPU),
			zap.Uint32("cpu_limit", projectLimits.CPULimit()),
			zap.Uint("exclude_container_id", excludeContainerID),
		)
		return containererrors.ErrProjectCPULimitExceeded
	}

	if newTotalMemory > projectLimits.MemoryLimit() {
		s.logger.Warn(ctx, "Project memory limit exceeded",
			zap.Uint("project_id", projectID),
			zap.Uint32("current_memory", totalMemory),
			zap.Uint32("requested_memory", requestedMemory),
			zap.Uint32("new_total_memory", newTotalMemory),
			zap.Uint32("memory_limit", projectLimits.MemoryLimit()),
			zap.Uint("exclude_container_id", excludeContainerID),
		)
		return containererrors.ErrProjectMemoryLimitExceeded
	}

	return nil
}
