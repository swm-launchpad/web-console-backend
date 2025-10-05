package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
)

func TestVolumeRepository_ConversionFunctions(t *testing.T) {
	// Note: volume repository uses the same conversion functions as project repository
	// so we don't need to test them again, but we can test volume-specific functionality

	t.Run("Volume reconstruction from database", func(t *testing.T) {
		// Test the ReconstructVolume functionality indirectly
		projectID := uint(123)
		name := "test-volume"
		capacity := uint32(1024)

		volume, err := model.NewVolume(projectID, name, capacity)
		require.NoError(t, err)

		// Verify the volume was created correctly
		assert.Equal(t, projectID, volume.ProjectID())
		assert.Equal(t, name, volume.Name())
		assert.Equal(t, capacity, volume.Capacity())
		assert.NotZero(t, volume.CreatedAt())
		// NewVolume sets updatedAt to zero time (NULL in database)
		assert.True(t, volume.UpdatedAt().IsZero())
	})

	t.Run("Volume with all fields", func(t *testing.T) {
		projectID := uint(456)
		name := "production-volume"
		capacity := uint32(2048)

		volume, err := model.NewVolume(projectID, name, capacity)
		require.NoError(t, err)

		volumeID := uint(789)
		volume.SetVolumeID(volumeID)

		// Verify all getters work correctly
		assert.Equal(t, volumeID, volume.VolumeID())
		assert.Equal(t, projectID, volume.ProjectID())
		assert.Equal(t, name, volume.Name())
		assert.Equal(t, capacity, volume.Capacity())
		assert.NotZero(t, volume.CreatedAt())
	})
}

func TestVolumeRepository_SQLCIntegration(t *testing.T) {
	t.Run("SQLC parameters conversion", func(t *testing.T) {
		// Test that we can create SQLC parameters from domain volume
		projectID := uint(123)
		name := "test-volume"
		capacity := uint32(1024)

		volume, err := model.NewVolume(projectID, name, capacity)
		require.NoError(t, err)
		volume.SetVolumeID(456)

		// Simulate creating SQLC parameters (like in the actual Create method)
		params := sqlc.CreateVolumeParams{
			ProjectID: uint32(volume.ProjectID()),
			Name:      volume.Name(),
			Capacity:  volume.Capacity(),
			CreatedAt: volume.CreatedAt(),
		}

		// Verify parameters are correct
		assert.Equal(t, uint32(123), params.ProjectID)
		assert.Equal(t, "test-volume", params.Name)
		assert.Equal(t, uint32(1024), params.Capacity)
		assert.NotZero(t, params.CreatedAt)
		// updated_at is omitted from INSERT, MySQL will use NULL default
	})

	t.Run("SQLC row to domain conversion", func(t *testing.T) {
		// Test converting from SQLC query result to domain model
		now := time.Now()

		// Simulate a database row result
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

		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
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

		volume := model.ReconstructVolume(
			uint(row.VolumeID),
			uint(row.ProjectID),
			row.Name,
			row.Capacity,
			row.CreatedAt,
			updatedAt,
		)

		// Verify reconstruction with null updated_at
		assert.Equal(t, uint(101), volume.VolumeID())
		assert.Equal(t, uint(202), volume.ProjectID())
		assert.Equal(t, "new-volume", volume.Name())
		assert.Equal(t, uint32(512), volume.Capacity())
		assert.Equal(t, now, volume.CreatedAt())
		assert.True(t, volume.UpdatedAt().IsZero()) // Should be zero value for null database value
	})
}

func TestVolumeRepository_ErrorHandling(t *testing.T) {
	t.Run("Invalid volume creation", func(t *testing.T) {
		// Test that invalid parameters are caught
		projectID := uint(0)  // Invalid project ID
		name := ""            // Invalid name
		capacity := uint32(0) // Invalid capacity

		_, err := model.NewVolume(projectID, name, capacity)
		assert.Error(t, err)
	})

	t.Run("Valid volume creation", func(t *testing.T) {
		// Test that valid parameters work
		projectID := uint(123)
		name := "valid-volume"
		capacity := uint32(1024)

		volume, err := model.NewVolume(projectID, name, capacity)
		require.NoError(t, err)

		assert.Equal(t, projectID, volume.ProjectID())
		assert.Equal(t, name, volume.Name())
		assert.Equal(t, capacity, volume.Capacity())
	})

	t.Run("Volume name validation", func(t *testing.T) {
		projectID := uint(123)
		capacity := uint32(1024)

		// Test invalid names
		invalidNames := []string{
			"",           // empty name
			"UPPERCASE",  // uppercase not allowed
			"with space", // spaces not allowed
			"with-UPPER", // mixed case not allowed
			"with_under", // underscore not allowed
			"123start",   // cannot start with number
			"-start",     // cannot start with hyphen
			"end-",       // cannot end with hyphen
		}

		for _, invalidName := range invalidNames {
			_, err := model.NewVolume(projectID, invalidName, capacity)
			assert.Error(t, err, "Expected error for invalid name: %s", invalidName)
		}

		// Test valid names - lowercase letters, numbers, hyphens; start with letter; end with letter or number
		validNames := []string{
			"a",          // single character is valid
			"ab",         // short name
			"valid-name", // kebab case
			"valid123",   // with numbers
			"v1-test-2",  // mixed
		}

		for _, validName := range validNames {
			volume, err := model.NewVolume(projectID, validName, capacity)
			assert.NoError(t, err, "Expected no error for valid name: %s", validName)
			if volume != nil {
				assert.Equal(t, validName, volume.Name())
			}
		}
	})

	t.Run("Volume capacity validation", func(t *testing.T) {
		projectID := uint(123)
		name := "test-volume"

		// Test invalid capacities
		invalidCapacities := []uint32{
			0,   // zero
			127, // below minimum
		}

		for _, invalidCapacity := range invalidCapacities {
			_, err := model.NewVolume(projectID, name, invalidCapacity)
			assert.Error(t, err, "Expected error for invalid capacity: %d", invalidCapacity)
		}

		// Test valid capacities
		validCapacities := []uint32{
			128,  // minimum
			1024, // typical
			2048, // maximum (MVP limit)
		}

		for _, validCapacity := range validCapacities {
			volume, err := model.NewVolume(projectID, name, validCapacity)
			assert.NoError(t, err, "Expected no error for valid capacity: %d", validCapacity)
			if volume != nil {
				assert.Equal(t, validCapacity, volume.Capacity())
			}
		}
	})
}
