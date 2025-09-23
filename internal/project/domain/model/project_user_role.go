package model

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ProjectUserRole represents the role of a user in a project as a value object
type ProjectUserRole string

const (
	ProjectUserRoleOwner ProjectUserRole = "owner"
)

// NewProjectUserRole creates a new ProjectUserRole with validation
func NewProjectUserRole(role string) (ProjectUserRole, error) {
	r := ProjectUserRole(role)
	if !r.IsValid() {
		return "", projecterrors.ErrInvalidUserRole
	}
	return r, nil
}

// IsValid checks if the role is a valid ProjectUserRole
func (r ProjectUserRole) IsValid() bool {
	return r == ProjectUserRoleOwner
}

// IsOwner checks if the role is owner
func (r ProjectUserRole) IsOwner() bool {
	return r == ProjectUserRoleOwner
}

// String returns the string representation of the role
func (r ProjectUserRole) String() string {
	return string(r)
}

// Equals checks if two ProjectUserRole values are equal
func (r ProjectUserRole) Equals(other ProjectUserRole) bool {
	return r == other
}
