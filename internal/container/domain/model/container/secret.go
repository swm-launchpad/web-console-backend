package model

import (
	"regexp"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// Secret represents a secret (sensitive environment variable) within the Container aggregate
// Secrets are stored in a separate table from regular environment variables
type Secret struct {
	secretID    uint
	containerID uint
	key         string
	value       string // TODO: Consider encryption for production
	createdAt   time.Time
	updatedAt   time.Time
}

const (
	MaxSecretKeyLength   = 255
	MaxSecretValueLength = 5000
)

var (
	// Secret key follows same naming convention as environment variables
	secretKeyRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

	// Reserved secret keys that should not be used
	reservedSecretKeys = map[string]bool{
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

// NewSecret creates a new Secret entity with validation
func NewSecret(containerID uint, key, value string) (*Secret, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	// Validate key
	if key == "" {
		return nil, containererrors.ErrSecretKeyRequired
	}
	if len(key) > MaxSecretKeyLength {
		return nil, containererrors.ErrSecretKeyTooLong
	}
	if !secretKeyRegex.MatchString(key) {
		return nil, containererrors.ErrInvalidSecretKey
	}
	if reservedSecretKeys[key] {
		return nil, containererrors.ErrReservedSecretKey
	}

	// Validate value
	if value == "" {
		return nil, containererrors.ErrSecretValueRequired
	}
	if len(value) > MaxSecretValueLength {
		return nil, containererrors.ErrSecretValueTooLong
	}

	return &Secret{
		containerID: containerID,
		key:         key,
		value:       value, // TODO: Encrypt in production
		createdAt:   time.Now(),
		updatedAt:   time.Time{}, // Zero time for new secrets (NULL in database)
	}, nil
}

// SecretID returns the secret ID
func (s *Secret) SecretID() uint {
	return s.secretID
}

// ContainerID returns the container ID
func (s *Secret) ContainerID() uint {
	return s.containerID
}

// Key returns the secret key
func (s *Secret) Key() string {
	return s.key
}

// Value returns the secret value
func (s *Secret) Value() string {
	return s.value
}

// CreatedAt returns the creation timestamp
func (s *Secret) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns the last update timestamp
func (s *Secret) UpdatedAt() time.Time {
	return s.updatedAt
}

// SetSecretID sets the secret ID (used by repository after persistence)
func (s *Secret) SetSecretID(id uint) {
	s.secretID = id
}

// UpdateValue updates the secret value
func (s *Secret) UpdateValue(newValue string) error {
	if newValue == "" {
		return containererrors.ErrSecretValueRequired
	}
	if len(newValue) > MaxSecretValueLength {
		return containererrors.ErrSecretValueTooLong
	}

	// No update if value is the same
	if s.value == newValue {
		return nil
	}

	s.value = newValue // TODO: Encrypt in production
	s.updatedAt = time.Now()
	return nil
}

// Equals checks if two Secrets have the same key
func (s *Secret) Equals(other *Secret) bool {
	if other == nil {
		return false
	}
	return s.key == other.key
}

// ReconstructSecret reconstructs a secret from persistence
// This is used when loading a secret from the database
func ReconstructSecret(
	secretID uint,
	containerID uint,
	key string,
	value string,
	createdAt time.Time,
	updatedAt time.Time,
) *Secret {
	return &Secret{
		secretID:    secretID,
		containerID: containerID,
		key:         key,
		value:       value, // TODO: Decrypt in production
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}
