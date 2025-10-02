package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

// PermissionService defines the interface for project permission checks
type PermissionService interface {
	// CanUserModifyProject checks if a user can modify (update/delete) a project
	CanUserModifyProject(ctx context.Context, userID uint, projectID uint) error

	// CanUserAccessProject checks if a user can access (read) a project
	CanUserAccessProject(ctx context.Context, userID uint, projectID uint) error

	// CanUserAddVolume checks if a user can add a volume to a project
	CanUserAddVolume(ctx context.Context, userID uint, projectID uint) error

	// CanUserRemoveVolume checks if a user can remove a volume
	CanUserRemoveVolume(ctx context.Context, userID uint, volumeID uint) error

	// CanUserAccessVolume checks if a user can access a volume
	CanUserAccessVolume(ctx context.Context, userID uint, volumeID uint) error
}

// permissionService is the concrete implementation of PermissionService
type permissionService struct {
	projectRepo repository.ProjectRepository
	volumeRepo  repository.VolumeRepository
}

// NewPermissionService creates a new instance of PermissionService
func NewPermissionService(projectRepo repository.ProjectRepository, volumeRepo repository.VolumeRepository) PermissionService {
	return &permissionService{
		projectRepo: projectRepo,
		volumeRepo:  volumeRepo,
	}
}

// CanUserModifyProject checks if a user can modify a project
func (s *permissionService) CanUserModifyProject(ctx context.Context, userID uint, projectID uint) error {
	if userID == 0 || projectID == 0 {
		return projecterrors.ErrPermissionDenied
	}

	// Get the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return err
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		return projecterrors.ErrPermissionDenied
	}

	// Check if user is an owner
	projectUser, err := project.GetUserByID(userID)
	if err != nil {
		return projecterrors.ErrPermissionDenied
	}

	if !projectUser.IsOwner() {
		return projecterrors.ErrOwnerRequired
	}

	return nil
}

// CanUserAccessProject checks if a user can access a project
func (s *permissionService) CanUserAccessProject(ctx context.Context, userID uint, projectID uint) error {
	if userID == 0 || projectID == 0 {
		return projecterrors.ErrPermissionDenied
	}

	// Get the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return err
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		return projecterrors.ErrPermissionDenied
	}

	return nil
}

// CanUserAddVolume checks if a user can add a volume to a project
func (s *permissionService) CanUserAddVolume(ctx context.Context, userID uint, projectID uint) error {
	// Adding a volume requires project modification permission
	return s.CanUserModifyProject(ctx, userID, projectID)
}

// CanUserRemoveVolume checks if a user can remove a volume
func (s *permissionService) CanUserRemoveVolume(ctx context.Context, userID uint, volumeID uint) error {
	if volumeID == 0 {
		return projecterrors.ErrVolumeNotFound
	}

	// Get the volume from repository
	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		return err
	}

	// Check if user can modify the project that owns this volume
	return s.CanUserModifyProject(ctx, userID, volume.ProjectID())
}

// CanUserAccessVolume checks if a user can access a volume
func (s *permissionService) CanUserAccessVolume(ctx context.Context, userID uint, volumeID uint) error {
	if volumeID == 0 {
		return projecterrors.ErrVolumeNotFound
	}

	// Get the volume from repository
	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		return err
	}

	// Check if user can access the project that owns this volume
	return s.CanUserAccessProject(ctx, userID, volume.ProjectID())
}
