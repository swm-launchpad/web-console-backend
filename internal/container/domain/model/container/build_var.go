package model

import (
	"regexp"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// BuildVar represents a build-time environment variable within the Container aggregate
// Build variables are used during the container build process (e.g., in Dockerfile)
// This is an entity within the Container aggregate (like EnvVar and Secret)
type BuildVar struct {
	buildVarID  uint
	containerID uint
	key         string
	value       string
	createdAt   time.Time
	updatedAt   time.Time
}

const (
	MaxBuildVarKeyLength   = 255
	MaxBuildVarValueLength = 5000
)

var (
	// Build variable key must start with uppercase letter or underscore,
	// followed by uppercase letters, numbers, or underscores
	buildVarKeyRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

	// Reserved build variable keys that should not be overridden
	reservedBuildVarKeys = map[string]bool{
		"PATH":     true,
		"HOME":     true,
		"USER":     true,
		"SHELL":    true,
		"PWD":      true,
		"LANG":     true,
		"TERM":     true,
		"HOSTNAME": true,
	}
)

// NewBuildVar creates a new BuildVar entity with validation
func NewBuildVar(containerID uint, key, value string) (*BuildVar, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	// Validate key
	if key == "" {
		return nil, containererrors.ErrBuildVarKeyRequired
	}
	if len(key) > MaxBuildVarKeyLength {
		return nil, containererrors.ErrBuildVarKeyTooLong
	}
	if !buildVarKeyRegex.MatchString(key) {
		return nil, containererrors.ErrInvalidBuildVarKey
	}
	if reservedBuildVarKeys[key] {
		return nil, containererrors.ErrReservedBuildVarKey
	}

	// Validate value
	if value == "" {
		return nil, containererrors.ErrBuildVarValueRequired
	}
	if len(value) > MaxBuildVarValueLength {
		return nil, containererrors.ErrBuildVarValueTooLong
	}

	return &BuildVar{
		containerID: containerID,
		key:         key,
		value:       value,
		createdAt:   time.Now(),
		updatedAt:   time.Time{}, // Zero time for new build vars (NULL in database)
	}, nil
}

// BuildVarID returns the build variable ID
func (b *BuildVar) BuildVarID() uint {
	return b.buildVarID
}

// ContainerID returns the container ID
func (b *BuildVar) ContainerID() uint {
	return b.containerID
}

// Key returns the build variable key
func (b *BuildVar) Key() string {
	return b.key
}

// Value returns the build variable value
func (b *BuildVar) Value() string {
	return b.value
}

// CreatedAt returns the creation timestamp
func (b *BuildVar) CreatedAt() time.Time {
	return b.createdAt
}

// UpdatedAt returns the last update timestamp
func (b *BuildVar) UpdatedAt() time.Time {
	return b.updatedAt
}

// SetBuildVarID sets the build variable ID (used by repository after persistence)
func (b *BuildVar) SetBuildVarID(id uint) {
	b.buildVarID = id
}

// UpdateValue updates the build variable value
func (b *BuildVar) UpdateValue(newValue string) error {
	if newValue == "" {
		return containererrors.ErrBuildVarValueRequired
	}
	if len(newValue) > MaxBuildVarValueLength {
		return containererrors.ErrBuildVarValueTooLong
	}

	// No update if value is the same
	if b.value == newValue {
		return nil
	}

	b.value = newValue
	b.updatedAt = time.Now()
	return nil
}

// Equals checks if two BuildVars have the same key
func (b *BuildVar) Equals(other *BuildVar) bool {
	if other == nil {
		return false
	}
	return b.key == other.key
}

// ReconstructBuildVar reconstructs a build variable from persistence
// This is used when loading a build var from the database
func ReconstructBuildVar(
	buildVarID uint,
	containerID uint,
	key string,
	value string,
	createdAt time.Time,
	updatedAt time.Time,
) *BuildVar {
	return &BuildVar{
		buildVarID:  buildVarID,
		containerID: containerID,
		key:         key,
		value:       value,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}
