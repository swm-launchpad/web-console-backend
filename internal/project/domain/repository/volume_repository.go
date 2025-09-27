package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
)

// VolumeRepository defines the interface for volume data persistence
// Volumes are their own aggregate root with independent lifecycle
type VolumeRepository interface {
	// Create persists a new volume and assigns its ID
	Create(ctx context.Context, volume *model.Volume) error

	// Save updates an existing volume
	Save(ctx context.Context, volume *model.Volume) error

	// FindByID retrieves a volume by its ID
	FindByID(ctx context.Context, volumeID uint) (*model.Volume, error)

	// FindByProjectID retrieves all volumes for a specific project
	FindByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error)

	// FindByName retrieves a volume by project ID and name
	FindByName(ctx context.Context, projectID uint, name string) (*model.Volume, error)

	// ExistsByName checks if a volume with the given name exists in a project
	ExistsByName(ctx context.Context, projectID uint, name string) (bool, error)

	// Delete removes a volume (physical delete since volumes contain data)
	Delete(ctx context.Context, volumeID uint) error

	// List retrieves volumes with pagination
	List(ctx context.Context, offset, limit int) ([]*model.Volume, error)

	// Count returns total number of volumes
	Count(ctx context.Context) (int64, error)

	// CountByProjectID returns total number of volumes for a project
	CountByProjectID(ctx context.Context, projectID uint) (int64, error)
}
