package value

import (
	"regexp"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ProjectSlug represents a URL-friendly project identifier
// It is a value object that encapsulates slug validation rules
type ProjectSlug struct {
	value string
}

const (
	// ProjectSlugLength is the fixed length for project slugs
	ProjectSlugLength = 23
)

var projectSlugRegex = regexp.MustCompile(`^p[0-9]{14}[a-z0-9]{8}$`)

// NewProjectSlug creates a new ProjectSlug with validation
// Validation rules:
// - Length: exactly 23 characters
// - Format: p{timestamp}{random} where timestamp is 14 digits (YYYYMMDDHHMMSS) and random is 8 alphanumeric chars
// - Example: p2025011812000012345678
func NewProjectSlug(slug string) (*ProjectSlug, error) {
	// Length validation
	if len(slug) != ProjectSlugLength {
		return nil, projecterrors.ErrSlugInvalidLength
	}

	// Format validation
	if !projectSlugRegex.MatchString(slug) {
		return nil, projecterrors.ErrSlugInvalidFormat
	}

	return &ProjectSlug{value: slug}, nil
}

// String returns the string representation of the slug
func (s ProjectSlug) String() string {
	return s.value
}

// Equals checks if two ProjectSlug values are equal
func (s ProjectSlug) Equals(other ProjectSlug) bool {
	return s.value == other.value
}
