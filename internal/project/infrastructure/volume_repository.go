package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/sqlc"
)

type volumeRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewVolumeRepository creates a new instance of VolumeRepository
func NewVolumeRepository(db sqlc.DBTX) repository.VolumeRepository {
	return &volumeRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// Create persists a new volume and assigns its ID
func (r *volumeRepository) Create(ctx context.Context, volume *model.Volume) error {
	qtx := r.queriesWithContext(ctx)

	// Create volume
	params := sqlc.CreateVolumeParams{
		ProjectID: uint32(volume.ProjectID()),
		Name:      volume.Name(),
		Capacity:  volume.Capacity(),
		CreatedAt: volume.CreatedAt(),
	}

	result, err := qtx.CreateVolume(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			return projecterrors.ErrDuplicateVolumeName
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Get the auto-generated ID
	volumeID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	volume.SetVolumeID(uint(volumeID))

	return nil
}

// FindByID retrieves a volume by its ID
func (r *volumeRepository) FindByID(ctx context.Context, volumeID uint) (*model.Volume, error) {
	qtx := r.queriesWithContext(ctx)

	row, err := qtx.GetVolumeByID(ctx, uint32(volumeID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrVolumeNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.toDomainVolume(row), nil
}

// FindByProjectID retrieves all volumes for a specific project
func (r *volumeRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	qtx := r.queriesWithContext(ctx)

	rows, err := qtx.GetVolumesByProjectID(ctx, uint32(projectID))
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	volumes := make([]*model.Volume, 0, len(rows))
	for _, row := range rows {
		volumes = append(volumes, r.toDomainVolume(row))
	}

	return volumes, nil
}

// FindByName retrieves a volume by project ID and name
func (r *volumeRepository) FindByName(ctx context.Context, projectID uint, name string) (*model.Volume, error) {
	qtx := r.queriesWithContext(ctx)

	params := sqlc.GetVolumeByNameParams{
		ProjectID: uint32(projectID),
		Name:      name,
	}

	row, err := qtx.GetVolumeByName(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrVolumeNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.toDomainVolume(row), nil
}

// ExistsByName checks if a volume with the given name exists in a project
func (r *volumeRepository) ExistsByName(ctx context.Context, projectID uint, name string) (bool, error) {
	qtx := r.queriesWithContext(ctx)

	params := sqlc.ExistsVolumeByNameParams{
		ProjectID: uint32(projectID),
		Name:      name,
	}

	exists, err := qtx.ExistsVolumeByName(ctx, params)
	if err != nil {
		return false, projecterrors.ErrDatabaseOperation
	}

	return exists, nil
}

// Delete removes a volume (physical delete since volumes contain data)
func (r *volumeRepository) Delete(ctx context.Context, volumeID uint) error {
	qtx := r.queriesWithContext(ctx)

	result, err := qtx.DeleteVolume(ctx, uint32(volumeID))
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}
	if rowsAffected == 0 {
		return projecterrors.ErrVolumeNotFound
	}

	return nil
}

// List retrieves volumes with pagination
func (r *volumeRepository) List(ctx context.Context, offset, limit int) ([]*model.Volume, error) {
	qtx := r.queriesWithContext(ctx)

	params := sqlc.ListVolumesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	rows, err := qtx.ListVolumes(ctx, params)
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	volumes := make([]*model.Volume, 0, len(rows))
	for _, row := range rows {
		volumes = append(volumes, r.toDomainVolume(row))
	}

	return volumes, nil
}

// Count returns total number of volumes
func (r *volumeRepository) Count(ctx context.Context) (int64, error) {
	qtx := r.queriesWithContext(ctx)

	count, err := qtx.CountVolumes(ctx)
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}

	return count, nil
}

// CountByProjectID returns total number of volumes for a project
func (r *volumeRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	qtx := r.queriesWithContext(ctx)

	count, err := qtx.CountVolumesByProjectID(ctx, uint32(projectID))
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}

	return count, nil
}

// DeleteByProjectID removes all volumes for a project (physical delete)
func (r *volumeRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	qtx := r.queriesWithContext(ctx)

	// Delete all volumes for the project
	_, err := qtx.DeleteVolumesByProjectID(ctx, uint32(projectID))
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	return nil
}

// GetTotalCapacityByProjectID returns total capacity of all volumes for a project
func (r *volumeRepository) GetTotalCapacityByProjectID(ctx context.Context, projectID uint) (uint32, error) {
	qtx := r.queriesWithContext(ctx)

	totalCapacity, err := qtx.GetTotalCapacityByProjectID(ctx, uint32(projectID))
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}

	// Handle nil case (no volumes)
	if totalCapacity == nil {
		return 0, nil
	}

	// Type assertion for the result - MySQL returns []uint8 for DECIMAL/SUM
	var capacity int64
	switch v := totalCapacity.(type) {
	case int64:
		capacity = v
	case []uint8:
		// MySQL returns DECIMAL as []uint8 (byte slice)
		// Convert to string and parse
		strVal := string(v)
		parsed, err := fmt.Sscanf(strVal, "%d", &capacity)
		if err != nil || parsed != 1 {
			return 0, projecterrors.ErrDatabaseOperation
		}
	default:
		return 0, projecterrors.ErrDatabaseOperation
	}

	return uint32(capacity), nil
}

// Helper methods

func (r *volumeRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	// Check if context has transaction
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *volumeRepository) toDomainVolume(row sqlc.Volume) *model.Volume {
	// Handle nullable updated_at
	var updatedAt time.Time
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	return model.ReconstructVolume(
		uint(row.VolumeID),
		uint(row.ProjectID),
		row.Name,
		row.Capacity,
		row.CreatedAt,
		updatedAt,
	)
}
