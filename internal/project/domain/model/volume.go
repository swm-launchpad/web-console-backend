package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Volume represents a storage volume aggregate root
// It has its own lifecycle independent of Project
type Volume struct {
	volumeID  uint
	projectID uint
	name      string
	capacity  uint32 // Capacity in GB
	createdAt time.Time
	updatedAt *time.Time
}

// Volume capacity limits in Mi (consistent with resource limits)
const (
	MinVolumeCapacity = 128   // 128Mi minimum
	MaxVolumeCapacity = 10240 // 10240Mi (~10GB) maximum
)

// NewVolume creates a new Volume aggregate root
func NewVolume(projectID uint, name string, capacity uint32) (*Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if name == "" {
		return nil, projecterrors.ErrVolumeNameRequired
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
		updatedAt: &now,
	}, nil
}

// GetVolumeID returns the volume ID
func (v *Volume) GetVolumeID() uint {
	return v.volumeID
}

// SetVolumeID sets the volume ID (typically set by repository after persistence)
func (v *Volume) SetVolumeID(id uint) {
	v.volumeID = id
}

// GetProjectID returns the project ID
func (v *Volume) GetProjectID() uint {
	return v.projectID
}

// GetName returns the volume name
func (v *Volume) GetName() string {
	return v.name
}

// GetCapacity returns the volume capacity in GB
func (v *Volume) GetCapacity() uint32 {
	return v.capacity
}

// GetCreatedAt returns the creation time
func (v *Volume) GetCreatedAt() time.Time {
	return v.createdAt
}

// GetUpdatedAt returns the last update time
func (v *Volume) GetUpdatedAt() *time.Time {
	if v.updatedAt == nil {
		return nil
	}
	t := *v.updatedAt
	return &t
}

// UpdateName updates the volume name
func (v *Volume) UpdateName(name string) error {
	if name == "" {
		return projecterrors.ErrVolumeNameRequired
	}
	v.name = name
	v.updateTimestamp()
	return nil
}

// UpdateCapacity updates the volume capacity
func (v *Volume) UpdateCapacity(capacity uint32) error {
	if capacity < MinVolumeCapacity {
		return projecterrors.ErrVolumeCapacityTooSmall
	}
	if capacity > MaxVolumeCapacity {
		return projecterrors.ErrVolumeCapacityExceeded
	}
	v.capacity = capacity
	v.updateTimestamp()
	return nil
}

// Update updates both name and capacity
func (v *Volume) Update(name string, capacity uint32) error {
	if name == "" {
		return projecterrors.ErrVolumeNameRequired
	}
	if capacity < MinVolumeCapacity {
		return projecterrors.ErrVolumeCapacityTooSmall
	}
	if capacity > MaxVolumeCapacity {
		return projecterrors.ErrVolumeCapacityExceeded
	}
	v.name = name
	v.capacity = capacity
	v.updateTimestamp()
	return nil
}

// Equals checks if two volumes are equal
func (v *Volume) Equals(other *Volume) bool {
	if other == nil {
		return false
	}
	return v.volumeID == other.volumeID &&
		v.projectID == other.projectID &&
		v.name == other.name &&
		v.capacity == other.capacity
}

// updateTimestamp updates the updated_at timestamp
func (v *Volume) updateTimestamp() {
	now := time.Now()
	v.updatedAt = &now
}

// ReconstructVolume reconstructs a volume from persistence
// This is used when loading a volume from the database
func ReconstructVolume(
	volumeID uint,
	projectID uint,
	name string,
	capacity uint32,
	createdAt time.Time,
	updatedAt *time.Time,
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
