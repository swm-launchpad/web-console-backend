package repository

import (
	"context"

	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
)

// ContainerRepository defines the interface for container data persistence
// This follows the repository pattern and is part of the domain layer
type ContainerRepository interface {
	// Create persists a new container and assigns its ID
	// The entire aggregate (including EnvVars, Networks, and Mounts) should be saved
	Create(ctx context.Context, container *model.Container) error

	// Save updates an existing container
	// The entire aggregate (including EnvVars, Networks, and Mounts) should be saved
	Save(ctx context.Context, container *model.Container) error

	// FindByID retrieves a container by its ID
	// Should return the complete aggregate with all EnvVars, Networks, and Mounts
	FindByID(ctx context.Context, containerID uint) (*model.Container, error)

	// FindByIDForUpdate retrieves a container by its ID with row lock (SELECT FOR UPDATE)
	// Used for preventing race conditions in concurrent modifications
	FindByIDForUpdate(ctx context.Context, containerID uint) (*model.Container, error)

	// FindByProjectID retrieves all containers for a specific project
	// Returns only containers that are not soft deleted
	FindByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error)

	// FindBySlug retrieves a container by slug
	// Slug is globally unique
	FindBySlug(ctx context.Context, slug string) (*model.Container, error)

	// ExistsBySlug checks if a container with the given slug exists
	// Used for slug uniqueness validation
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// ExistsByNameAndProjectID checks if a container with the given name exists in the project
	// Used for name uniqueness validation within a project
	ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error)

	// Delete soft deletes a container (sets is_deleted = true)
	Delete(ctx context.Context, containerID uint) error

	// DeleteByProjectID soft deletes all containers for a project
	// Used when cascading project deletion
	DeleteByProjectID(ctx context.Context, projectID uint) error

	// List retrieves containers with pagination
	List(ctx context.Context, offset, limit int) ([]*model.Container, error)

	// Count returns total number of non-deleted containers
	Count(ctx context.Context) (int64, error)

	// CountByProjectID returns total number of non-deleted containers for a project
	CountByProjectID(ctx context.Context, projectID uint) (int64, error)

	// CountByTemplateID returns total number of containers using a specific template
	// Useful for template deletion validation
	CountByTemplateID(ctx context.Context, templateID uint) (int64, error)

	// GetTotalResourceUsageByProject calculates the total resource usage for a project
	// Returns total CPU (millicores) and memory (Mi) used by all active containers in the project
	// Used for validating against project resource limits
	GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (totalCPU uint32, totalMemory uint32, err error)
}
