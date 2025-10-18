package value

import (
	"regexp"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// VolumeSlug represents a URL-friendly volume identifier
// It is a value object that encapsulates slug validation rules
type VolumeSlug struct {
	value string
}

const (
	// VolumeSlugLength is the fixed length for volume slugs
	VolumeSlugLength = 23
)

var volumeSlugRegex = regexp.MustCompile(`^v[0-9]{14}[a-z0-9]{8}$`)

// NewVolumeSlug creates a new VolumeSlug with validation
// Validation rules:
// - Length: exactly 23 characters
// - Format: v{timestamp}{random} where timestamp is 14 digits (YYYYMMDDHHMMSS) and random is 8 alphanumeric chars
// - Example: v2025011812000012345678
func NewVolumeSlug(slug string) (VolumeSlug, error) {
	// Length validation
	if len(slug) != VolumeSlugLength {
		return VolumeSlug{}, projecterrors.ErrSlugInvalidLength
	}

	// Format validation
	if !volumeSlugRegex.MatchString(slug) {
		return VolumeSlug{}, projecterrors.ErrSlugInvalidFormat
	}

	return VolumeSlug{value: slug}, nil
}

// String returns the string representation of the slug
func (s VolumeSlug) String() string {
	return s.value
}

// Equals checks if two slugs are equal
func (s VolumeSlug) Equals(other VolumeSlug) bool {
	return s.value == other.value
}
