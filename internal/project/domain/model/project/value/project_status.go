package value

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ProjectStatus represents the status of a project as a value object
type ProjectStatus string

const (
	ProjectStatusActive ProjectStatus = "active"
)

// NewProjectStatus creates a new ProjectStatus with validation
func NewProjectStatus(status string) (ProjectStatus, error) {
	ps := ProjectStatus(status)
	if !ps.isValid() {
		return "", projecterrors.ErrInvalidFormat
	}
	return ps, nil
}

// isValid checks if the status is a valid ProjectStatus
func (s ProjectStatus) isValid() bool {
	return s == ProjectStatusActive
}

// IsActive checks if the status represents an active state
func (s ProjectStatus) IsActive() bool {
	return s == ProjectStatusActive
}

// String returns the string representation of the status
func (s ProjectStatus) String() string {
	return string(s)
}

// Equals checks if two ProjectStatus values are equal
func (s ProjectStatus) Equals(other ProjectStatus) bool {
	return s == other
}
