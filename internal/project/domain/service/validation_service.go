package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// ValidationService defines the interface for project validation logic
type ValidationService interface {
	// ValidateFreeResources validates that Free plan resources match the fixed values
	ValidateFreeResources(plan value.Plan, limits value.ResourceLimits) error

	// ValidateFreeTierLimits validates resources against free tier limits (beta period)
	ValidateFreeTierLimits(plan value.Plan, limits value.ResourceLimits) error

	// ValidateFreePlanLimit validates that user doesn't exceed Free plan project limit (1 per user)
	ValidateFreePlanLimit(ctx context.Context, userID uint, plan value.Plan) error
}

// validationService is the concrete implementation of ValidationService
type validationService struct {
	projectRepo     repository.ProjectRepository
	settingsService settings.SettingsService
}

// NewValidationService creates a new instance of ValidationService
func NewValidationService(projectRepo repository.ProjectRepository, settingsService settings.SettingsService) ValidationService {
	return &validationService{
		projectRepo:     projectRepo,
		settingsService: settingsService,
	}
}

// ValidateFreeResources validates that Free plan resources match the fixed values
// Free plan has fixed resources from database settings
func (s *validationService) ValidateFreeResources(plan value.Plan, limits value.ResourceLimits) error {
	if !plan.HasFixedResources() {
		return nil // Not a Free plan, no validation needed
	}

	// Get Free plan limits from database
	cpuLimit, err := s.settingsService.GetFreePlanCPULimit()
	if err != nil {
		return err
	}
	memoryLimit, err := s.settingsService.GetFreePlanMemoryLimit()
	if err != nil {
		return err
	}
	diskLimit, err := s.settingsService.GetFreePlanDiskLimit()
	if err != nil {
		return err
	}

	// Free plan must have exact resource values
	if limits.CPULimit() != uint32(cpuLimit) ||
		limits.MemoryLimit() != uint32(memoryLimit) ||
		limits.DiskLimit() != uint32(diskLimit) {
		return projecterrors.ErrFreeResourcesFixed
	}

	return nil
}

// ValidateFreeTierLimits validates resources against free tier limits (beta period)
// Beta tier limits are retrieved from database settings
// Free plan is exempt as it has fixed resources
func (s *validationService) ValidateFreeTierLimits(plan value.Plan, limits value.ResourceLimits) error {
	// Check if beta tier is enabled
	enabled, err := s.settingsService.IsBetaTierEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return nil // Free tier limits are disabled
	}

	if plan.HasFixedResources() {
		return nil // Free plan already validated by ValidateFreeResources
	}

	// Get beta tier limits from database
	cpuLimit, err := s.settingsService.GetBetaTierCPULimit()
	if err != nil {
		return err
	}
	memoryLimit, err := s.settingsService.GetBetaTierMemoryLimit()
	if err != nil {
		return err
	}
	diskLimit, err := s.settingsService.GetBetaTierDiskLimit()
	if err != nil {
		return err
	}

	// Check CPU limit
	if limits.CPULimit() > uint32(cpuLimit) {
		return projecterrors.ErrFreeTierResourceExceeded
	}

	// Check memory limit
	if limits.MemoryLimit() > uint32(memoryLimit) {
		return projecterrors.ErrFreeTierResourceExceeded
	}

	// Check disk limit
	if limits.DiskLimit() > uint32(diskLimit) {
		return projecterrors.ErrFreeTierResourceExceeded
	}

	return nil
}

// ValidateFreePlanLimit validates that user doesn't exceed Free plan project limit
// Free plan project limit is retrieved from database settings
func (s *validationService) ValidateFreePlanLimit(ctx context.Context, userID uint, plan value.Plan) error {
	if plan != value.PlanFree {
		return nil // Not a Free plan, no limit
	}

	// Get Free plan max projects from database
	maxProjects, err := s.settingsService.GetFreePlanMaxProjects()
	if err != nil {
		return err
	}

	// Count existing Free plan projects for this user
	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Count active Free plan projects
	freeCount := 0
	for _, project := range projects {
		plan, hasPlan := project.Plan()
		if hasPlan && plan == value.PlanFree && project.Status() == value.ProjectStatusActive {
			freeCount++
		}
	}

	if freeCount >= maxProjects {
		return projecterrors.ErrFreePlanLimitExceeded
	}

	return nil
}
