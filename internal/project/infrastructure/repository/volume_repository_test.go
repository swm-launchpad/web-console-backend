package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
)

func TestVolumeRepository_ConversionFunctions(t *testing.T) {
	// Note: volume repository uses the same conversion functions as project repository
	// so we don't need to test them again, but we can test volume-specific functionality

	t.Run("Volume reconstruction from database", func(t *testing.T) {
		// Test the ReconstructVolume functionality indirectly
		// This mimics how the repository converts database rows to domain objects

		// Simulate a database row (from SQLC generated code)
		now := time.Now()

		row := sqlc.Volume{
			VolumeID:  789,
			ProjectID: 123,
			Name:      "db-volume",
			Capacity:  2048,
			CreatedAt: now,
			UpdatedAt: sql.NullTime{Time: now.Add(time.Hour), Valid: true},
		}

		// Use ReconstructVolume like the actual repository does
		var updatedAt time.Time
		if row.UpdatedAt.Valid {
			updatedAt = row.UpdatedAt.Time
		}

		testSlug, _ := value.NewVolumeSlug("v2025011812000012345678")
		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
			testSlug,
			row.Capacity,
			row.CreatedAt,
			updatedAt,
		)

		// Verify reconstruction
		assert.Equal(t, uint(789), volume.VolumeID())
		assert.Equal(t, uint(123), volume.ProjectID())
		assert.Equal(t, "db-volume", volume.Name())
		assert.Equal(t, uint32(2048), volume.Capacity())
		assert.Equal(t, now, volume.CreatedAt())
		assert.False(t, volume.UpdatedAt().IsZero())
		assert.Equal(t, now.Add(time.Hour), volume.UpdatedAt())
	})

	t.Run("SQLC row with null updated_at", func(t *testing.T) {
		// Test converting from SQLC query result with null updated_at
		now := time.Now()

		row := sqlc.Volume{
			VolumeID:  101,
			ProjectID: 202,
			Name:      "new-volume",
			Capacity:  512,
			CreatedAt: now,
			UpdatedAt: sql.NullTime{Valid: false}, // NULL updated_at
		}

		// Use the same pattern as repository: check Valid before accessing Time
		var updatedAt time.Time
		if row.UpdatedAt.Valid {
			updatedAt = row.UpdatedAt.Time
		}

		testSlug, _ := value.NewVolumeSlug("v2025011812000012345678")
		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
			testSlug,
			row.Capacity,
			row.CreatedAt,
			updatedAt,
		)

		// Verify that updatedAt is zero time when NULL
		assert.Equal(t, uint(101), volume.VolumeID())
		assert.Equal(t, uint(202), volume.ProjectID())
		assert.True(t, volume.UpdatedAt().IsZero(), "UpdatedAt should be zero time when null in database")
	})
}

// Note: Integration tests should cover full CRUD operations
// Unit tests focus on the conversion logic that's unique to volume repository
