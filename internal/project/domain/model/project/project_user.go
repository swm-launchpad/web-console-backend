package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// ProjectUser represents a user's membership in a project
// This is an entity within the Project aggregate
type ProjectUser struct {
	projectUserID uint
	projectID     uint
	userID        uint
	role          value.ProjectUserRole
	isDeleted     bool
	deletedAt     *time.Time
	createdAt     time.Time
	updatedAt     time.Time
}

// NewProjectUser creates a new ProjectUser entity
func NewProjectUser(projectID, userID uint, role value.ProjectUserRole) (*ProjectUser, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if userID == 0 {
		return nil, projecterrors.ErrInvalidUserID
	}

	now := time.Now()
	return &ProjectUser{
		projectID: projectID,
		userID:    userID,
		role:      role,
		isDeleted: false,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ProjectUserID returns the project user ID
func (pu *ProjectUser) ProjectUserID() uint {
	return pu.projectUserID
}

// ProjectID returns the project ID
func (pu *ProjectUser) ProjectID() uint {
	return pu.projectID
}

// UserID returns the user ID
func (pu *ProjectUser) UserID() uint {
	return pu.userID
}

// Role returns the user's role in the project
func (pu *ProjectUser) Role() value.ProjectUserRole {
	return pu.role
}

// CreatedAt returns the creation time
func (pu *ProjectUser) CreatedAt() time.Time {
	return pu.createdAt
}

// UpdatedAt returns the last update time
func (pu *ProjectUser) UpdatedAt() time.Time {
	return pu.updatedAt
}

// IsDeleted returns whether the user is soft deleted from the project
func (pu *ProjectUser) IsDeleted() bool {
	return pu.isDeleted
}

// DeletedAt returns the deletion time
func (pu *ProjectUser) DeletedAt() *time.Time {
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
func (pu *ProjectUser) ChangeRole(newRole value.ProjectUserRole) error {
	if pu.isDeleted {
		return projecterrors.ErrUserNotInProject
	}

	if pu.role == newRole {
		// No change needed
		return nil
	}

	pu.role = newRole
	pu.updatedAt = time.Now()

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
	pu.updatedAt = now

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
	pu.updatedAt = now

	return nil
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
