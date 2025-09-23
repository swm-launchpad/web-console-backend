package model

import (
	"regexp"
	"strings"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ProjectSlug represents a validated project slug as a value object
type ProjectSlug struct {
	value string
}

const (
	slugMinLength = 3
	slugMaxLength = 63
)

var (
	// slugRegex validates slug format: lowercase letters, numbers, and hyphens
	// Must start with a letter, cannot end with hyphen, no consecutive hyphens
	slugRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

// NewProjectSlug creates a new ProjectSlug with validation
func NewProjectSlug(slug string) (*ProjectSlug, error) {
	// Convert to lowercase
	slug = strings.ToLower(strings.TrimSpace(slug))

	// Check if empty
	if slug == "" {
		return nil, projecterrors.ErrSlugRequired
	}

	// Check length constraints
	if len(slug) < slugMinLength {
		return nil, projecterrors.ErrSlugTooShort
	}
	if len(slug) > slugMaxLength {
		return nil, projecterrors.ErrSlugTooLong
	}

	// Check format
	if !slugRegex.MatchString(slug) {
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

// IsEmpty checks if the slug is empty
func (s ProjectSlug) IsEmpty() bool {
	return s.value == ""
}
