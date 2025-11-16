package repository

import (
	"context"
	"time"

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

	// CheckInternalPortExistsInProject checks if an internal port is already used by another container in the same project
	// Used for validating network port uniqueness within a project (K8s pod shares network interface)
	CheckInternalPortExistsInProject(ctx context.Context, projectID uint, internalPort uint16) (bool, error)

	// CheckFQDNExists checks if an FQDN is used anywhere in the system
	// Returns true if the FQDN exists in any non-deleted network
	// Used for simple FQDN availability check across all projects
	CheckFQDNExists(ctx context.Context, fqdn string) (bool, error)

	// CheckFQDNExistsForProject checks FQDN with proper business rules
	// Returns true if FQDN exists in:
	// 1. Same project (any active network, regardless of container state) - prevents duplicates
	// 2. Other projects (only if container is active) - maintains ownership
	// Allows reuse when:
	// - Same project soft-deleted network (deployment time same, no conflict)
	// - Other project soft-deleted container (ownership released)
	// Used for AddNetwork and CreateContainer
	CheckFQDNExistsForProject(ctx context.Context, fqdn string, projectID uint) (bool, error)

	// CheckFQDNExistsForProjectExcludingSelf same as CheckFQDNExistsForProject but excludes self
	// Used for UpdateNetwork to allow updating network's own FQDN
	CheckFQDNExistsForProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error)

	// CheckInternalPortExistsInProjectExcludingSelf checks internal port with self-exclusion
	// Returns true if the internal port exists in the project, excluding the specified network
	// Used for UpdateNetwork to allow updating a network's internal port
	CheckInternalPortExistsInProjectExcludingSelf(ctx context.Context, projectID uint, internalPort uint16, networkID uint) (bool, error)

	// SoftDeleteNetworksByContainerID soft deletes all networks for a container
	// Used when a container is deleted to cascade soft delete to networks
	// Preserves FQDN ownership tracking even after container deletion
	SoftDeleteNetworksByContainerID(ctx context.Context, containerID uint) error

	// SoftDeleteNetworkByID soft deletes a single network by network ID
	// Used when FQDN is changed to preserve old FQDN ownership tracking
	// Preserves FQDN value in database for ownership tracking
	SoftDeleteNetworkByID(ctx context.Context, networkID uint) error

	// FindAllSlugsByProjectIDIncludingDeleted retrieves all container slugs for a project
	// INCLUDING soft-deleted containers. Used for cleanup operations that need to remove
	// all resources including those from deleted containers.
	FindAllSlugsByProjectIDIncludingDeleted(ctx context.Context, projectID uint) ([]string, error)

	// UpdateNetworkTektonEventID updates the Tekton PipelineRun name for a network
	// and clears external_ip, external_port, expires_at (used when starting new NodePort creation)
	UpdateNetworkTektonEventID(ctx context.Context, networkID uint, tektonEventID string) error

	// UpdateNetworkNodePortResult updates the NodePort result fields (external_ip, external_port, expires_at)
	// Used when PipelineRun completes and NodePort information becomes available
	UpdateNetworkNodePortResult(ctx context.Context, networkID uint, externalIP string, externalPort uint16, expiresAt time.Time) error
}
