package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
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
	// Check if we're already in a transaction
	var shouldCommit bool
	tx, existingTx := db.GetTx(ctx)
	if !existingTx || tx == nil {
		// No existing transaction, create our own
		var err error
		tx, err = r.beginTx(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()
		shouldCommit = true
	}

	qtx := r.queriesWithContext(ctx, tx)

	// Create volume
	params := sqlc.CreateVolumeParams{
		ProjectID: uint32(volume.GetProjectID()),
		Name:      volume.GetName(),
		Capacity:  volume.GetCapacity(),
		CreatedAt: volume.GetCreatedAt(),
		UpdatedAt: toNullTime(volume.GetUpdatedAt()),
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

	// Commit if this is our transaction
	if shouldCommit {
		if err := tx.Commit(); err != nil {
			return projecterrors.ErrDatabaseOperation
		}
	}

	return nil
}

// Save updates an existing volume
func (r *volumeRepository) Save(ctx context.Context, volume *model.Volume) error {
	// Check if we're already in a transaction
	var shouldCommit bool
	tx, existingTx := db.GetTx(ctx)
	if !existingTx || tx == nil {
		// No existing transaction, create our own
		var err error
		tx, err = r.beginTx(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()
		shouldCommit = true
	}

	qtx := r.queriesWithContext(ctx, tx)

	// Update volume
	params := sqlc.UpdateVolumeParams{
		Name:      volume.GetName(),
		Capacity:  volume.GetCapacity(),
		UpdatedAt: toNullTime(volume.GetUpdatedAt()),
		VolumeID:  uint32(volume.GetVolumeID()),
	}

	result, err := qtx.UpdateVolume(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			return projecterrors.ErrDuplicateVolumeName
		}
		return projecterrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return projecterrors.ErrVolumeNotFound
	}

	// Commit if this is our transaction
	if shouldCommit {
		if err := tx.Commit(); err != nil {
			return projecterrors.ErrDatabaseOperation
		}
	}

	return nil
}

// FindByID retrieves a volume by its ID
func (r *volumeRepository) FindByID(ctx context.Context, volumeID uint) (*model.Volume, error) {
	qtx := r.queriesWithContext(ctx, r.db)

	row, err := qtx.GetVolumeByID(ctx, uint32(volumeID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrVolumeNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Reconstruct volume from database
	volume := model.ReconstructVolume(
		uint(row.VolumeID),
		uint(row.ProjectID),
		row.Name,
		row.Capacity,
		row.CreatedAt,
		fromNullTime(row.UpdatedAt),
	)

	return volume, nil
}

// FindByProjectID retrieves all volumes for a specific project
func (r *volumeRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	qtx := r.queriesWithContext(ctx, r.db)

	rows, err := qtx.GetVolumesByProjectID(ctx, uint32(projectID))
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	volumes := make([]*model.Volume, 0, len(rows))
	for _, row := range rows {
		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
			row.Capacity,
			row.CreatedAt,
			fromNullTime(row.UpdatedAt),
		)
		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// FindByName retrieves a volume by project ID and name
func (r *volumeRepository) FindByName(ctx context.Context, projectID uint, name string) (*model.Volume, error) {
	qtx := r.queriesWithContext(ctx, r.db)

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

	// Reconstruct volume from database
	volume := model.ReconstructVolume(
		uint(row.VolumeID),
		uint(row.ProjectID),
		row.Name,
		row.Capacity,
		row.CreatedAt,
		fromNullTime(row.UpdatedAt),
	)

	return volume, nil
}

// ExistsByName checks if a volume with the given name exists in a project
func (r *volumeRepository) ExistsByName(ctx context.Context, projectID uint, name string) (bool, error) {
	qtx := r.queriesWithContext(ctx, r.db)

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
	// Check if we're already in a transaction
	var shouldCommit bool
	tx, existingTx := db.GetTx(ctx)
	if !existingTx || tx == nil {
		// No existing transaction, create our own
		var err error
		tx, err = r.beginTx(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()
		shouldCommit = true
	}

	qtx := r.queriesWithContext(ctx, tx)

	result, err := qtx.DeleteVolume(ctx, uint32(volumeID))
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return projecterrors.ErrVolumeNotFound
	}

	// Commit if this is our transaction
	if shouldCommit {
		if err := tx.Commit(); err != nil {
			return projecterrors.ErrDatabaseOperation
		}
	}

	return nil
}

// List retrieves volumes with pagination
func (r *volumeRepository) List(ctx context.Context, offset, limit int) ([]*model.Volume, error) {
	qtx := r.queriesWithContext(ctx, r.db)

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
		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
			row.Capacity,
			row.CreatedAt,
			fromNullTime(row.UpdatedAt),
		)
		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// Count returns total number of volumes
func (r *volumeRepository) Count(ctx context.Context) (int64, error) {
	qtx := r.queriesWithContext(ctx, r.db)

	count, err := qtx.CountVolumes(ctx)
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}

	return count, nil
}

// CountByProjectID returns total number of volumes for a project
func (r *volumeRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	qtx := r.queriesWithContext(ctx, r.db)

	count, err := qtx.CountVolumesByProjectID(ctx, uint32(projectID))
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}

	return count, nil
}

// Helper methods

func (r *volumeRepository) beginTx(ctx context.Context) (*sql.Tx, error) {
	sqlDB, ok := r.db.(*sql.DB)
	if !ok {
		return nil, errors.New("database does not support transactions")
	}
	return sqlDB.BeginTx(ctx, nil)
}

func (r *volumeRepository) queriesWithContext(_ context.Context, db sqlc.DBTX) *sqlc.Queries {
	// If db is nil, use the default queries
	if db == nil {
		return r.queries
	}
	return sqlc.New(db)
}
