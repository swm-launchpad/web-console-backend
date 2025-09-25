package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Volume represents a storage volume in a project
// This is an entity within the Project aggregate
type Volume struct {
	volumeID  uint
	projectID uint
	name      string
	capacity  uint32 // Capacity in GB
	createdAt time.Time
	updatedAt *time.Time
}

// NewVolume creates a new Volume entity
func NewVolume(projectID uint, name string, capacity uint32) (*Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if name == "" {
		return nil, projecterrors.ErrVolumeNameRequired
	}
	if capacity == 0 {
		return nil, projecterrors.ErrInvalidCapacity
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

// setVolumeID sets the volume ID (for use by repository and tests only)
func (v *Volume) setVolumeID(id uint) {
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
	if capacity == 0 {
		return projecterrors.ErrInvalidCapacity
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
	if capacity == 0 {
		return projecterrors.ErrInvalidCapacity
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
