package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ProjectUser represents a user's membership in a project
// This is an entity within the Project aggregate
type ProjectUser struct {
	projectUserID uint
	projectID     uint
	userID        uint
	role          ProjectUserRole
	isDeleted     bool
	deletedAt     *time.Time
	createdAt     time.Time
	updatedAt     *time.Time
}

// NewProjectUser creates a new ProjectUser entity
func NewProjectUser(projectID, userID uint, role ProjectUserRole) (*ProjectUser, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if userID == 0 {
		return nil, projecterrors.ErrInvalidUserID
	}
	if !role.IsValid() {
		return nil, projecterrors.ErrInvalidUserRole
	}

	now := time.Now()
	return &ProjectUser{
		projectID: projectID,
		userID:    userID,
		role:      role,
		isDeleted: false,
		createdAt: now,
		updatedAt: &now,
	}, nil
}

// GetProjectUserID returns the project user ID
func (pu *ProjectUser) GetProjectUserID() uint {
	return pu.projectUserID
}

// GetProjectID returns the project ID
func (pu *ProjectUser) GetProjectID() uint {
	return pu.projectID
}

// GetUserID returns the user ID
func (pu *ProjectUser) GetUserID() uint {
	return pu.userID
}

// GetRole returns the user's role in the project
func (pu *ProjectUser) GetRole() ProjectUserRole {
	return pu.role
}

// GetCreatedAt returns the creation time
func (pu *ProjectUser) GetCreatedAt() time.Time {
	return pu.createdAt
}

// GetUpdatedAt returns the last update time
func (pu *ProjectUser) GetUpdatedAt() *time.Time {
	if pu.updatedAt == nil {
		return nil
	}
	t := *pu.updatedAt
	return &t
}

// IsDeleted returns whether the user is soft deleted from the project
func (pu *ProjectUser) IsDeleted() bool {
	return pu.isDeleted
}

// GetDeletedAt returns the deletion time
func (pu *ProjectUser) GetDeletedAt() *time.Time {
	if pu.deletedAt == nil {
		return nil
	}
	t := *pu.deletedAt
	return &t
}

// SetProjectUserID sets the project user ID (typically set by repository after persistence)
func (pu *ProjectUser) SetProjectUserID(id uint) {
	pu.projectUserID = id
}

// ChangeRole changes the user's role in the project
func (pu *ProjectUser) ChangeRole(newRole ProjectUserRole) error {
	if !newRole.IsValid() {
		return projecterrors.ErrInvalidUserRole
	}

	if pu.isDeleted {
		return projecterrors.ErrUserNotInProject
	}

	if pu.role == newRole {
		// No change needed
		return nil
	}

	pu.role = newRole
	now := time.Now()
	pu.updatedAt = &now

	return nil
}

// IsOwner checks if the user is an owner of the project
func (pu *ProjectUser) IsOwner() bool {
	return !pu.isDeleted && pu.role.IsOwner()
}

// SoftDelete soft deletes the user from the project
func (pu *ProjectUser) SoftDelete() error {
	if pu.isDeleted {
		return nil // Already deleted
	}

	pu.isDeleted = true
	now := time.Now()
	pu.deletedAt = &now
	pu.updatedAt = &now

	return nil
}

// Restore restores a soft deleted user
func (pu *ProjectUser) Restore() error {
	if !pu.isDeleted {
		return nil // Not deleted
	}

	pu.isDeleted = false
	pu.deletedAt = nil
	now := time.Now()
	pu.updatedAt = &now

	return nil
}

// Equals checks if two ProjectUser entities are equal
func (pu *ProjectUser) Equals(other *ProjectUser) bool {
	if other == nil {
		return false
	}

	return pu.projectID == other.projectID &&
		pu.userID == other.userID &&
		pu.role.Equals(other.role) &&
		pu.isDeleted == other.isDeleted
}

// BelongsToProject checks if the user belongs to a specific project
func (pu *ProjectUser) BelongsToProject(projectID uint) bool {
	return pu.projectID == projectID
}

// BelongsToUser checks if this entity belongs to a specific user
func (pu *ProjectUser) BelongsToUser(userID uint) bool {
	return pu.userID == userID
}

// IsActive checks if the user is active in the project (not deleted)
func (pu *ProjectUser) IsActive() bool {
	return !pu.isDeleted
}
