package model

import (
	"regexp"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// EnvVar represents an environment variable within the Container aggregate
// This is an entity within the Container aggregate (like ProjectUser in Project)
type EnvVar struct {
	envVarID    uint
	containerID uint
	key         string
	value       string
	createdAt   time.Time
	updatedAt   time.Time
}

const (
	MaxEnvVarKeyLength   = 255
	MaxEnvVarValueLength = 5000
)

var (
	// Environment variable key must start with uppercase letter or underscore,
	// followed by uppercase letters, numbers, or underscores
	envVarKeyRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

	// Reserved environment variables that should not be overridden
	reservedEnvVarKeys = map[string]bool{
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

// NewEnvVar creates a new EnvVar entity with validation
func NewEnvVar(containerID uint, key, value string) (*EnvVar, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	// Validate key
	if key == "" {
		return nil, containererrors.ErrEnvVarKeyRequired
	}
	if len(key) > MaxEnvVarKeyLength {
		return nil, containererrors.ErrEnvVarKeyTooLong
	}
	if !envVarKeyRegex.MatchString(key) {
		return nil, containererrors.ErrInvalidEnvVarKey
	}
	if reservedEnvVarKeys[key] {
		return nil, containererrors.ErrReservedEnvVarKey
	}

	// Validate value
	if value == "" {
		return nil, containererrors.ErrEnvVarValueRequired
	}
	if len(value) > MaxEnvVarValueLength {
		return nil, containererrors.ErrEnvVarValueTooLong
	}

	return &EnvVar{
		containerID: containerID,
		key:         key,
		value:       value,
		createdAt:   time.Now(),
		updatedAt:   time.Time{}, // Zero time for new env vars (NULL in database)
	}, nil
}

// EnvVarID returns the environment variable ID
func (e *EnvVar) EnvVarID() uint {
	return e.envVarID
}

// ContainerID returns the container ID
func (e *EnvVar) ContainerID() uint {
	return e.containerID
}

// Key returns the environment variable key
func (e *EnvVar) Key() string {
	return e.key
}

// Value returns the environment variable value
func (e *EnvVar) Value() string {
	return e.value
}

// CreatedAt returns the creation timestamp
func (e *EnvVar) CreatedAt() time.Time {
	return e.createdAt
}

// UpdatedAt returns the last update timestamp
func (e *EnvVar) UpdatedAt() time.Time {
	return e.updatedAt
}

// SetEnvVarID sets the environment variable ID (used by repository after persistence)
func (e *EnvVar) SetEnvVarID(id uint) {
	e.envVarID = id
}

// UpdateValue updates the environment variable value
func (e *EnvVar) UpdateValue(newValue string) error {
	if newValue == "" {
		return containererrors.ErrEnvVarValueRequired
	}
	if len(newValue) > MaxEnvVarValueLength {
		return containererrors.ErrEnvVarValueTooLong
	}

	// No update if value is the same
	if e.value == newValue {
		return nil
	}

	e.value = newValue
	e.updatedAt = time.Now()
	return nil
}

// Equals checks if two EnvVars have the same key
func (e *EnvVar) Equals(other *EnvVar) bool {
	if other == nil {
		return false
	}
	return e.key == other.key
}

// ReconstructEnvVar reconstructs an environment variable from persistence
// This is used when loading an env var from the database
func ReconstructEnvVar(
	envVarID uint,
	containerID uint,
	key string,
	value string,
	createdAt time.Time,
	updatedAt time.Time,
) *EnvVar {
	return &EnvVar{
		envVarID:    envVarID,
		containerID: containerID,
		key:         key,
		value:       value,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}
