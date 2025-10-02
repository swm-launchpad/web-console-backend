package model

import (
	"regexp"
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Volume represents a storage volume aggregate root
// It has its own lifecycle independent of Project
type Volume struct {
	volumeID  uint
	projectID uint
	name      string
	capacity  uint32 // Capacity in Mi (Mebibytes)
	createdAt time.Time
	updatedAt time.Time
}

// Volume capacity limits in Mi (consistent with resource limits)
const (
	MinVolumeCapacity = 128  // 128Mi minimum
	MaxVolumeCapacity = 2048 // 2048Mi (2GiB) maximum - Business rule: volume capacity limit
)

// volumeNamePattern defines the valid volume name format
// - Must start with a lowercase letter
// - Can contain lowercase letters, numbers, and hyphens
// - Must end with a lowercase letter or number
var volumeNamePattern = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)

// NewVolume creates a new Volume aggregate root
func NewVolume(projectID uint, name string, capacity uint32) (*Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if name == "" {
		return nil, projecterrors.ErrVolumeNameRequired
	}
	if !volumeNamePattern.MatchString(name) {
		return nil, projecterrors.ErrInvalidVolumeName
	}
	if capacity < MinVolumeCapacity {
		return nil, projecterrors.ErrVolumeCapacityTooSmall
	}
	if capacity > MaxVolumeCapacity {
		return nil, projecterrors.ErrVolumeCapacityExceeded
	}

	now := time.Now()
	return &Volume{
		volumeID:  0, // Will be set by repository
		projectID: projectID,
		name:      name,
		capacity:  capacity,
		createdAt: now,
		updatedAt: time.Time{}, // Zero time for new volumes (NULL in database)
	}, nil
}

// VolumeID returns the volume ID
func (v *Volume) VolumeID() uint {
	return v.volumeID
}

// ProjectID returns the project ID
func (v *Volume) ProjectID() uint {
	return v.projectID
}

// Name returns the volume name
func (v *Volume) Name() string {
	return v.name
}

// Capacity returns the volume capacity in Mi (Mebibytes)
func (v *Volume) Capacity() uint32 {
	return v.capacity
}

// CreatedAt returns the creation time
func (v *Volume) CreatedAt() time.Time {
	return v.createdAt
}

// UpdatedAt returns the last update time
func (v *Volume) UpdatedAt() time.Time {
	return v.updatedAt
}

// SetVolumeID sets the volume ID (typically set by repository after persistence)
func (v *Volume) SetVolumeID(id uint) {
	v.volumeID = id
}

// ReconstructVolume reconstructs a volume from persistence
// This is used when loading a volume from the database
func ReconstructVolume(
	volumeID uint,
	projectID uint,
	name string,
	capacity uint32,
	createdAt time.Time,
	updatedAt time.Time,
) *Volume {
	return &Volume{
		volumeID:  volumeID,
		projectID: projectID,
		name:      name,
		capacity:  capacity,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}
