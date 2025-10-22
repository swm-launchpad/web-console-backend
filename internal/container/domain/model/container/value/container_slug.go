package value

import (
	"regexp"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// ContainerSlug represents a URL-friendly container identifier
// It is a value object that encapsulates slug validation rules
type ContainerSlug struct {
	value string
}

const (
	// ContainerSlugLength is the fixed length for container slugs
	ContainerSlugLength = 23
)

var containerSlugRegex = regexp.MustCompile(`^c[0-9]{14}[a-z0-9]{8}$`)

// NewContainerSlug creates a new ContainerSlug with validation
// Validation rules:
// - Length: exactly 23 characters
// - Format: c{timestamp}{random} where timestamp is 14 digits (YYYYMMDDHHMMSS) and random is 8 alphanumeric chars
// - Example: c2025011812000012345678
func NewContainerSlug(slug string) (ContainerSlug, error) {
	// Length validation
	if len(slug) != ContainerSlugLength {
		return ContainerSlug{}, containererrors.ErrSlugInvalidLength
	}

	// Format validation
	if !containerSlugRegex.MatchString(slug) {
		return ContainerSlug{}, containererrors.ErrSlugInvalidFormat
	}

	return ContainerSlug{value: slug}, nil
}

// String returns the string representation of the slug
func (s ContainerSlug) String() string {
	return s.value
}

// Equals checks if two slugs are equal
func (s ContainerSlug) Equals(other ContainerSlug) bool {
	return s.value == other.value
}
