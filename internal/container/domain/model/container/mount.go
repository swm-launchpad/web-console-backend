package model

import (
	"strings"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// Mount represents a volume mount within the Container aggregate
// This is an entity within the Container aggregate that links containers to volumes
type Mount struct {
	containerID uint
	volumeID    uint
	mountPath   string
	createdAt   time.Time
	updatedAt   time.Time
}

const (
	MaxMountPathLength = 255
)

// NewMount creates a new Mount entity with validation
func NewMount(containerID, volumeID uint, mountPath string) (*Mount, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	if volumeID == 0 {
		return nil, containererrors.ErrInvalidVolumeID
	}

	// Validate mount path
	if mountPath == "" {
		return nil, containererrors.ErrMountPathRequired
	}
	if len(mountPath) > MaxMountPathLength {
		return nil, containererrors.ErrMountPathTooLong
	}
	if !strings.HasPrefix(mountPath, "/") {
		return nil, containererrors.ErrInvalidMountFormat
	}

	// Prevent mounting to system directories
	if isSystemPath(mountPath) {
		return nil, containererrors.ErrMountPathReserved
	}

	return &Mount{
		containerID: containerID,
		volumeID:    volumeID,
		mountPath:   mountPath,
		createdAt:   time.Now(),
		updatedAt:   time.Time{}, // Zero time for new mounts (NULL in database)
	}, nil
}

// ContainerID returns the container ID
func (m *Mount) ContainerID() uint {
	return m.containerID
}

// VolumeID returns the volume ID
func (m *Mount) VolumeID() uint {
	return m.volumeID
}

// MountPath returns the mount path
func (m *Mount) MountPath() string {
	return m.mountPath
}

// CreatedAt returns the creation timestamp
func (m *Mount) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m *Mount) UpdatedAt() time.Time {
	return m.updatedAt
}

// UpdateMountPath updates the mount path
func (m *Mount) UpdateMountPath(newPath string) error {
	if newPath == "" {
		return containererrors.ErrMountPathRequired
	}
	if len(newPath) > MaxMountPathLength {
		return containererrors.ErrMountPathTooLong
	}
	if !strings.HasPrefix(newPath, "/") {
		return containererrors.ErrInvalidMountFormat
	}
	if isSystemPath(newPath) {
		return containererrors.ErrMountPathReserved
	}

	// No update if path is the same
	if m.mountPath == newPath {
		return nil
	}

	m.mountPath = newPath
	m.updatedAt = time.Now()
	return nil
}

// Equals checks if two Mounts refer to the same volume
func (m *Mount) Equals(other *Mount) bool {
	if other == nil {
		return false
	}
	return m.volumeID == other.volumeID
}

// isSystemPath checks if the path is a reserved system directory
func isSystemPath(path string) bool {
	// System directories that should not be mounted
	systemPaths := []string{
		"/bin", "/boot", "/dev", "/etc", "/lib", "/lib64",
		"/proc", "/root", "/run", "/sbin", "/sys", "/usr",
	}

	for _, sysPath := range systemPaths {
		if path == sysPath || strings.HasPrefix(path, sysPath+"/") {
			return true
		}
	}
	return false
}

// ReconstructMount reconstructs a mount from persistence
// This is used when loading a mount from the database
func ReconstructMount(
	containerID uint,
	volumeID uint,
	mountPath string,
	createdAt time.Time,
	updatedAt time.Time,
) *Mount {
	return &Mount{
		containerID: containerID,
		volumeID:    volumeID,
		mountPath:   mountPath,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}
