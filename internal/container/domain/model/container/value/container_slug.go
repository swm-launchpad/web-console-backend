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

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// NewContainerSlug creates a new ContainerSlug with validation
// Validation rules:
// - Length: 3-63 characters
// - Format: lowercase letters, numbers, and hyphens only
// - Must start and end with lowercase letter or number
// - Cannot contain consecutive hyphens
func NewContainerSlug(slug string) (ContainerSlug, error) {
	// Length validation
	if len(slug) < 3 {
		return ContainerSlug{}, containererrors.ErrSlugTooShort
	}
	if len(slug) > 63 {
		return ContainerSlug{}, containererrors.ErrSlugTooLong
	}

	// Single character edge case
	if len(slug) == 1 {
		if !regexp.MustCompile(`^[a-z0-9]$`).MatchString(slug) {
			return ContainerSlug{}, containererrors.ErrSlugInvalidFormat
		}
		return ContainerSlug{value: slug}, nil
	}

	// Format validation
	if !slugRegex.MatchString(slug) {
		return ContainerSlug{}, containererrors.ErrSlugInvalidFormat
	}

	// Check for consecutive hyphens
	if regexp.MustCompile(`--`).MatchString(slug) {
		return ContainerSlug{}, containererrors.ErrSlugInvalidFormat
	}

	// Check reserved slugs
	if isReservedSlug(slug) {
		return ContainerSlug{}, containererrors.ErrSlugReserved
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

// isReservedSlug checks if the slug is reserved
var reservedSlugs = map[string]bool{
	"api":       true,
	"admin":     true,
	"www":       true,
	"ftp":       true,
	"mail":      true,
	"smtp":      true,
	"pop":       true,
	"imap":      true,
	"localhost": true,
	"system":    true,
	"root":      true,
	"default":   true,
}

func isReservedSlug(slug string) bool {
	return reservedSlugs[slug]
}
