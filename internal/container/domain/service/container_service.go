package service

import (
	"context"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// ContainerService defines the interface for container-related business logic
type ContainerService interface {
	// CreateContainer creates a new container with the given parameters
	// gitConfig and resourceLimits are required, templateID and templateConfig are optional
	// slug is automatically generated from name
	CreateContainer(ctx context.Context, projectID uint, name string, gitConfig value.GitConfig, resourceLimits value.ResourceLimits, templateID *uint, templateConfig map[string]interface{}) (*model.Container, error)

	// GetContainer retrieves a container by ID
	GetContainer(ctx context.Context, containerID uint) (*model.Container, error)

	// UpdateContainer updates an existing container
	UpdateContainer(ctx context.Context, containerID uint, updateFn func(*model.Container) error) (*model.Container, error)

	// DeleteContainer soft deletes a container
	DeleteContainer(ctx context.Context, containerID uint) error

	// ListContainersByProjectID retrieves all containers for a project
	ListContainersByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error)

	// CountContainersByProjectID returns the number of active containers for a project
	CountContainersByProjectID(ctx context.Context, projectID uint) (int, error)

	// CheckContainerNameExists checks if a container name already exists in a project
	CheckContainerNameExists(ctx context.Context, projectID uint, name string) (bool, error)
}

// containerService is the concrete implementation of ContainerService
type containerService struct {
	containerRepo repository.ContainerRepository
	slugService   SlugService
}

// NewContainerService creates a new instance of ContainerService
func NewContainerService(containerRepo repository.ContainerRepository, slugService SlugService) ContainerService {
	return &containerService{
		containerRepo: containerRepo,
		slugService:   slugService,
	}
}

// CreateContainer creates a new container with validation
// slug is automatically generated from name
// gitConfig and resourceLimits are required, templateID and templateConfig are optional
func (s *containerService) CreateContainer(ctx context.Context, projectID uint, name string, gitConfig value.GitConfig, resourceLimits value.ResourceLimits, templateID *uint, templateConfig map[string]interface{}) (*model.Container, error) {
	if projectID == 0 {
		return nil, containererrors.ErrInvalidProjectID
	}

	// Check if container name already exists in this project
	exists, err := s.CheckContainerNameExists(ctx, projectID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, containererrors.ErrContainerNameExists
	}

	// Generate slug from name
	slug, err := s.slugService.GenerateSlugFromName(ctx, projectID, name)
	if err != nil {
		return nil, err
	}

	// Create the container aggregate with all fields
	container, err := model.NewContainer(projectID, name, slug, gitConfig, resourceLimits, templateID, templateConfig)
	if err != nil {
		return nil, err
	}

	// Persist the container
	if err := s.containerRepo.Create(ctx, container); err != nil {
		return nil, containererrors.ErrContainerCreationFailed
	}

	return container, nil
}

// GetContainer retrieves a container by ID
func (s *containerService) GetContainer(ctx context.Context, containerID uint) (*model.Container, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	container, err := s.containerRepo.FindByID(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return container, nil
}

// UpdateContainer updates an existing container
func (s *containerService) UpdateContainer(ctx context.Context, containerID uint, updateFn func(*model.Container) error) (*model.Container, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	// Retrieve the container
	container, err := s.containerRepo.FindByID(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Apply the update function
	if err := updateFn(container); err != nil {
		return nil, err
	}

	// Save the updated container
	if err := s.containerRepo.Save(ctx, container); err != nil {
		return nil, err
	}

	return container, nil
}

// DeleteContainer soft deletes a container
func (s *containerService) DeleteContainer(ctx context.Context, containerID uint) error {
	if containerID == 0 {
		return containererrors.ErrInvalidContainerID
	}

	// Retrieve the container
	container, err := s.containerRepo.FindByID(ctx, containerID)
	if err != nil {
		return err
	}

	// Soft delete the container
	if err := container.SoftDelete(); err != nil {
		return err
	}

	// Save the deleted state
	if err := s.containerRepo.Save(ctx, container); err != nil {
		return err
	}

	return nil
}

// ListContainersByProjectID retrieves all containers for a project
func (s *containerService) ListContainersByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error) {
	if projectID == 0 {
		return nil, containererrors.ErrInvalidProjectID
	}

	containers, err := s.containerRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

// CountContainersByProjectID returns the number of active containers for a project
func (s *containerService) CountContainersByProjectID(ctx context.Context, projectID uint) (int, error) {
	if projectID == 0 {
		return 0, containererrors.ErrInvalidProjectID
	}

	count, err := s.containerRepo.CountByProjectID(ctx, projectID)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// CheckContainerNameExists checks if a container name already exists in a project
func (s *containerService) CheckContainerNameExists(ctx context.Context, projectID uint, name string) (bool, error) {
	if projectID == 0 {
		return false, containererrors.ErrInvalidProjectID
	}
	if name == "" {
		return false, containererrors.ErrNameRequired
	}

	exists, err := s.containerRepo.ExistsByNameAndProjectID(ctx, projectID, name)
	if err != nil {
		return false, err
	}

	return exists, nil
}
