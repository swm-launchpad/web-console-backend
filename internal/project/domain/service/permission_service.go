package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
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
	logger      logger.Logger
}

// NewPermissionService creates a new instance of PermissionService
func NewPermissionService(projectRepo repository.ProjectRepository, volumeRepo repository.VolumeRepository, log logger.Logger) PermissionService {
	return &permissionService{
		projectRepo: projectRepo,
		volumeRepo:  volumeRepo,
		logger:      log,
	}
}

// CanUserModifyProject checks if a user can modify a project
func (s *permissionService) CanUserModifyProject(ctx context.Context, userID uint, projectID uint) error {
	s.logger.Info(ctx, "can user modify project started",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
	)

	if userID == 0 || projectID == 0 {
		s.logger.Error(ctx, "permission denied (invalid IDs)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrPermissionDenied
	}

	// Get the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		if err == projecterrors.ErrProjectNotFound {
			return err
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		s.logger.Error(ctx, "permission denied (user not in project)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrPermissionDenied
	}

	// Check if user is an owner
	projectUser, err := project.GetUserByID(userID)
	if err != nil {
		s.logger.Error(ctx, "permission denied (failed to get project user)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return projecterrors.ErrPermissionDenied
	}

	if !projectUser.IsOwner() {
		s.logger.Error(ctx, "permission denied (owner required)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrOwnerRequired
	}

	s.logger.Info(ctx, "can user modify project completed (allowed)",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
	)
	return nil
}

// CanUserAccessProject checks if a user can access a project
func (s *permissionService) CanUserAccessProject(ctx context.Context, userID uint, projectID uint) error {
	s.logger.Info(ctx, "can user access project started",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
	)

	if userID == 0 || projectID == 0 {
		s.logger.Error(ctx, "permission denied (invalid IDs)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrPermissionDenied
	}

	// Get the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		if err == projecterrors.ErrProjectNotFound {
			return err
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		s.logger.Error(ctx, "permission denied (user not in project)",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrPermissionDenied
	}

	s.logger.Info(ctx, "can user access project completed (allowed)",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
	)
	return nil
}

// CanUserAddVolume checks if a user can add a volume to a project
func (s *permissionService) CanUserAddVolume(ctx context.Context, userID uint, projectID uint) error {
	// Adding a volume requires project modification permission
	return s.CanUserModifyProject(ctx, userID, projectID)
}

// CanUserRemoveVolume checks if a user can remove a volume
func (s *permissionService) CanUserRemoveVolume(ctx context.Context, userID uint, volumeID uint) error {
	s.logger.Info(ctx, "can user remove volume started",
		zap.Uint("user_id", userID),
		zap.Uint("volume_id", volumeID),
	)

	if volumeID == 0 {
		s.logger.Error(ctx, "volume not found (invalid ID)",
			zap.Uint("volume_id", volumeID),
		)
		return projecterrors.ErrVolumeNotFound
	}

	// Get the volume from repository
	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		s.logger.Error(ctx, "failed to find volume",
			zap.Uint("user_id", userID),
			zap.Uint("volume_id", volumeID),
			zap.Error(err),
		)
		return err
	}

	// Check if user can modify the project that owns this volume
	return s.CanUserModifyProject(ctx, userID, volume.ProjectID())
}

// CanUserAccessVolume checks if a user can access a volume
func (s *permissionService) CanUserAccessVolume(ctx context.Context, userID uint, volumeID uint) error {
	s.logger.Info(ctx, "can user access volume started",
		zap.Uint("user_id", userID),
		zap.Uint("volume_id", volumeID),
	)

	if volumeID == 0 {
		s.logger.Error(ctx, "volume not found (invalid ID)",
			zap.Uint("volume_id", volumeID),
		)
		return projecterrors.ErrVolumeNotFound
	}

	// Get the volume from repository
	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		s.logger.Error(ctx, "failed to find volume",
			zap.Uint("user_id", userID),
			zap.Uint("volume_id", volumeID),
			zap.Error(err),
		)
		return err
	}

	// Check if user can access the project that owns this volume
	return s.CanUserAccessProject(ctx, userID, volume.ProjectID())
}
