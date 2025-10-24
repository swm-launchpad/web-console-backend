package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// PermissionService defines the interface for container permission checks
type PermissionService interface {
	// CanUserModifyContainer checks if a user can modify (update/delete) a container
	CanUserModifyContainer(ctx context.Context, userID uint, containerID uint) error

	// CanUserAccessContainer checks if a user can access (read) a container
	CanUserAccessContainer(ctx context.Context, userID uint, containerID uint) error

	// CanUserCreateContainer checks if a user can create a container in a project
	CanUserCreateContainer(ctx context.Context, userID uint, projectID uint) error

	// CanUserManageEnvVars checks if a user can manage environment variables
	CanUserManageEnvVars(ctx context.Context, userID uint, containerID uint) error

	// CanUserManageNetworks checks if a user can manage network port mappings
	CanUserManageNetworks(ctx context.Context, userID uint, containerID uint) error

	// CanUserManageMounts checks if a user can manage volume mounts
	CanUserManageMounts(ctx context.Context, userID uint, containerID uint) error
}

// permissionService is the concrete implementation of PermissionService
type permissionService struct {
	containerRepo repository.ContainerRepository
	projectRepo   projectrepo.ProjectRepository
	logger        logger.Logger
}

// NewPermissionService creates a new instance of PermissionService
func NewPermissionService(
	containerRepo repository.ContainerRepository,
	projectRepo projectrepo.ProjectRepository,
	log logger.Logger,
) PermissionService {
	return &permissionService{
		containerRepo: containerRepo,
		projectRepo:   projectRepo,
		logger:        log,
	}
}

// CanUserModifyContainer checks if a user can modify a container
func (s *permissionService) CanUserModifyContainer(ctx context.Context, userID uint, containerID uint) error {
	if userID == 0 || containerID == 0 {
		return containererrors.ErrPermissionDenied
	}

	// Get the container
	container, err := s.containerRepo.FindByID(ctx, containerID)
	if err != nil {
		if err == containererrors.ErrContainerNotFound {
			return err
		}
		s.logger.Error(ctx, "Failed to find container for modify permission check",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	// Get the project that owns this container
	project, err := s.projectRepo.FindByID(ctx, container.ProjectID())
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return containererrors.ErrInvalidProjectID
		}
		s.logger.Error(ctx, "Failed to find project for modify permission check",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		s.logger.Warn(ctx, "User not in project - modify permission denied",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
		)
		return containererrors.ErrPermissionDenied
	}

	// Check if user is an owner
	projectUser, err := project.GetUserByID(userID)
	if err != nil {
		s.logger.Warn(ctx, "User not found in project - modify permission denied",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
		)
		return containererrors.ErrPermissionDenied
	}

	if !projectUser.IsOwner() {
		s.logger.Warn(ctx, "User is not project owner - owner required for modification",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
			zap.String("user_role", string(projectUser.Role())),
		)
		return containererrors.ErrOwnerRequired
	}

	return nil
}

// CanUserAccessContainer checks if a user can access a container
func (s *permissionService) CanUserAccessContainer(ctx context.Context, userID uint, containerID uint) error {
	if userID == 0 || containerID == 0 {
		return containererrors.ErrPermissionDenied
	}

	// Get the container
	container, err := s.containerRepo.FindByID(ctx, containerID)
	if err != nil {
		if err == containererrors.ErrContainerNotFound {
			return err
		}
		s.logger.Error(ctx, "Failed to find container for access permission check",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	// Get the project that owns this container
	project, err := s.projectRepo.FindByID(ctx, container.ProjectID())
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return containererrors.ErrInvalidProjectID
		}
		s.logger.Error(ctx, "Failed to find project for access permission check",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		s.logger.Warn(ctx, "User not in project - access permission denied",
			zap.Uint("user_id", userID),
			zap.Uint("container_id", containerID),
			zap.Uint("project_id", container.ProjectID()),
		)
		return containererrors.ErrPermissionDenied
	}

	return nil
}

// CanUserCreateContainer checks if a user can create a container in a project
func (s *permissionService) CanUserCreateContainer(ctx context.Context, userID uint, projectID uint) error {
	if userID == 0 || projectID == 0 {
		return containererrors.ErrPermissionDenied
	}

	// Get the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return containererrors.ErrInvalidProjectID
		}
		s.logger.Error(ctx, "Failed to find project for create permission check",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	// Check if user is in the project
	if !project.HasUser(userID) {
		s.logger.Warn(ctx, "User not in project - create permission denied",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return containererrors.ErrPermissionDenied
	}

	// Check if user is an owner
	projectUser, err := project.GetUserByID(userID)
	if err != nil {
		s.logger.Warn(ctx, "User not found in project - create permission denied",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
		)
		return containererrors.ErrPermissionDenied
	}

	if !projectUser.IsOwner() {
		s.logger.Warn(ctx, "User is not project owner - owner required for creation",
			zap.Uint("user_id", userID),
			zap.Uint("project_id", projectID),
			zap.String("user_role", string(projectUser.Role())),
		)
		return containererrors.ErrOwnerRequired
	}

	return nil
}

// CanUserManageEnvVars checks if a user can manage environment variables
func (s *permissionService) CanUserManageEnvVars(ctx context.Context, userID uint, containerID uint) error {
	// Managing env vars requires container modification permission
	return s.CanUserModifyContainer(ctx, userID, containerID)
}

// CanUserManageNetworks checks if a user can manage network port mappings
func (s *permissionService) CanUserManageNetworks(ctx context.Context, userID uint, containerID uint) error {
	// Managing networks requires container modification permission
	return s.CanUserModifyContainer(ctx, userID, containerID)
}

// CanUserManageMounts checks if a user can manage volume mounts
func (s *permissionService) CanUserManageMounts(ctx context.Context, userID uint, containerID uint) error {
	// Managing mounts requires container modification permission
	return s.CanUserModifyContainer(ctx, userID, containerID)
}
