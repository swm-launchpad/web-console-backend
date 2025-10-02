package value

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

	ps := &ProjectSlug{value: slug}
	if err := ps.validate(); err != nil {
		return nil, err
	}

	return ps, nil
}

// validate validates the slug format and constraints
func (s *ProjectSlug) validate() error {
	// Check if empty
	if s.value == "" {
		return projecterrors.ErrSlugRequired
	}

	// Check length constraints
	if len(s.value) < slugMinLength {
		return projecterrors.ErrSlugTooShort
	}
	if len(s.value) > slugMaxLength {
		return projecterrors.ErrSlugTooLong
	}

	// Check format
	if !slugRegex.MatchString(s.value) {
		return projecterrors.ErrSlugInvalidFormat
	}

	return nil
}

// String returns the string representation of the slug
func (s ProjectSlug) String() string {
	return s.value
}

// Equals checks if two ProjectSlug values are equal
func (s ProjectSlug) Equals(other ProjectSlug) bool {
	return s.value == other.value
}
