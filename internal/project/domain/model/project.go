package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Project represents a project aggregate root
// It manages ProjectUser entities and enforces domain invariants
type Project struct {
	projectID uint
	name      string
	slug      ProjectSlug
	fqdn      *string
	status    ProjectStatus
	plan      *string
	limits    ResourceLimits
	users     []ProjectUser // Aggregate's internal entities
	volumes   []Volume      // Aggregate's internal entities
	isDeleted bool
	deletedAt *time.Time
	createdAt time.Time
	updatedAt *time.Time
}

// NewProject creates a new project with an initial owner
func NewProject(name string, slug ProjectSlug, ownerID uint) (*Project, error) {
	if name == "" {
		return nil, projecterrors.ErrNameRequired
	}
	if slug.IsEmpty() {
		return nil, projecterrors.ErrSlugRequired
	}
	if ownerID == 0 {
		return nil, projecterrors.ErrOwnerIDRequired
	}

	now := time.Now()
	project := &Project{
		name:      name,
		slug:      slug,
		status:    ProjectStatusActive,
		limits:    ResourceLimits{}, // Default unlimited
		users:     make([]ProjectUser, 0),
		volumes:   make([]Volume, 0),
		isDeleted: false,
		createdAt: now,
		updatedAt: &now,
	}

	// Add the initial owner without projectID (will be set later)
	// Create ProjectUser directly to avoid validation error
	owner := &ProjectUser{
		projectID: 0, // Will be set when project ID is assigned
		userID:    ownerID,
		role:      ProjectUserRoleOwner,
		isDeleted: false,
		createdAt: now,
		updatedAt: &now,
	}
	project.users = append(project.users, *owner)

	return project, nil
}

// GetProjectID returns the project ID
func (p *Project) GetProjectID() uint {
	return p.projectID
}

// SetProjectID sets the project ID (typically set by repository after persistence)
func (p *Project) SetProjectID(id uint) {
	p.projectID = id
	// Update projectID for all users
	for i := range p.users {
		p.users[i].projectID = id
	}
	// Update projectID for all volumes
	for i := range p.volumes {
		p.volumes[i].projectID = id
	}
}

// GetName returns the project name
func (p *Project) GetName() string {
	return p.name
}

// GetSlug returns the project slug
func (p *Project) GetSlug() ProjectSlug {
	return p.slug
}

// GetFQDN returns the fully qualified domain name
func (p *Project) GetFQDN() *string {
	if p.fqdn == nil {
		return nil
	}
	s := *p.fqdn
	return &s
}

// GetStatus returns the project status
func (p *Project) GetStatus() ProjectStatus {
	return p.status
}

// GetPlan returns the project plan
func (p *Project) GetPlan() *string {
	if p.plan == nil {
		return nil
	}
	s := *p.plan
	return &s
}

// GetLimits returns the resource limits
func (p *Project) GetLimits() ResourceLimits {
	return p.limits
}

// GetUsers returns a copy of project users
func (p *Project) GetUsers() []ProjectUser {
	users := make([]ProjectUser, len(p.users))
	copy(users, p.users)
	return users
}

// GetActiveUsers returns only active (non-deleted) users
func (p *Project) GetActiveUsers() []ProjectUser {
	var activeUsers []ProjectUser
	for _, user := range p.users {
		if user.IsActive() {
			activeUsers = append(activeUsers, user)
		}
	}
	return activeUsers
}

// GetCreatedAt returns the creation time
func (p *Project) GetCreatedAt() time.Time {
	return p.createdAt
}

// GetUpdatedAt returns the last update time
func (p *Project) GetUpdatedAt() *time.Time {
	if p.updatedAt == nil {
		return nil
	}
	t := *p.updatedAt
	return &t
}

// IsDeleted returns whether the project is soft deleted
func (p *Project) IsDeleted() bool {
	return p.isDeleted
}

// GetDeletedAt returns the deletion time
func (p *Project) GetDeletedAt() *time.Time {
	if p.deletedAt == nil {
		return nil
	}
	t := *p.deletedAt
	return &t
}

// AddUser adds a new user to the project
func (p *Project) AddUser(userID uint, role ProjectUserRole) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Check if user already exists
	for i := range p.users {
		if p.users[i].GetUserID() == userID {
			if p.users[i].IsDeleted() {
				// Restore the user with new role
				if err := p.users[i].Restore(); err != nil {
					return err
				}
				return p.users[i].ChangeRole(role)
			}
			return projecterrors.ErrUserAlreadyInProject
		}
	}

	// Create new project user
	projectUser, err := NewProjectUser(p.projectID, userID, role)
	if err != nil {
		return err
	}

	p.users = append(p.users, *projectUser)
	p.updateTimestamp()

	return nil
}

// RemoveUser removes a user from the project
func (p *Project) RemoveUser(userID uint) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Find the user
	userIndex := -1
	for i := range p.users {
		if p.users[i].GetUserID() == userID && p.users[i].IsActive() {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return projecterrors.ErrUserNotInProject
	}

	// Check if this is the last owner
	if p.users[userIndex].IsOwner() && !p.hasOtherOwner(userID) {
		return projecterrors.ErrCannotRemoveLastOwner
	}

	// Soft delete the user
	_ = p.users[userIndex].SoftDelete() // Error ignored as SoftDelete always succeeds
	p.updateTimestamp()

	return nil
}

// ChangeUserRole changes a user's role in the project
func (p *Project) ChangeUserRole(userID uint, newRole ProjectUserRole) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Find the user
	userIndex := -1
	for i := range p.users {
		if p.users[i].GetUserID() == userID && p.users[i].IsActive() {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return projecterrors.ErrUserNotInProject
	}

	// If changing from owner to another role, check if there's another owner
	if p.users[userIndex].IsOwner() && !newRole.IsOwner() && !p.hasOtherOwner(userID) {
		return projecterrors.ErrCannotRemoveLastOwner
	}

	// Change the role
	if err := p.users[userIndex].ChangeRole(newRole); err != nil {
		return err
	}

	p.updateTimestamp()
	return nil
}

// GetUserByID returns a user by their ID
func (p *Project) GetUserByID(userID uint) (*ProjectUser, error) {
	for i := range p.users {
		if p.users[i].GetUserID() == userID && p.users[i].IsActive() {
			user := p.users[i]
			return &user, nil
		}
	}
	return nil, projecterrors.ErrUserNotInProject
}

// HasUser checks if a user is in the project
func (p *Project) HasUser(userID uint) bool {
	for _, user := range p.users {
		if user.GetUserID() == userID && user.IsActive() {
			return true
		}
	}
	return false
}

// GetOwners returns all owners of the project
func (p *Project) GetOwners() []ProjectUser {
	var owners []ProjectUser
	for _, user := range p.users {
		if user.IsOwner() && user.IsActive() {
			owners = append(owners, user)
		}
	}
	return owners
}

// HasOwner checks if the project has at least one owner
func (p *Project) HasOwner() bool {
	for _, user := range p.users {
		if user.IsOwner() && user.IsActive() {
			return true
		}
	}
	return false
}

// AddVolume adds a new volume to the project
func (p *Project) AddVolume(name string, capacity uint32) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Check for duplicate volume name
	for _, v := range p.volumes {
		if v.GetName() == name {
			return projecterrors.ErrDuplicateVolumeName
		}
	}

	// Create new volume
	volume, err := NewVolume(p.projectID, name, capacity)
	if err != nil {
		return err
	}

	p.volumes = append(p.volumes, *volume)
	p.updateTimestamp()
	return nil
}

// RemoveVolume removes a volume from the project
func (p *Project) RemoveVolume(volumeID uint) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Find and remove the volume
	for i, v := range p.volumes {
		if v.GetVolumeID() == volumeID {
			// Remove volume from slice
			p.volumes = append(p.volumes[:i], p.volumes[i+1:]...)
			p.updateTimestamp()
			return nil
		}
	}

	return projecterrors.ErrVolumeNotFound
}

// UpdateVolume updates an existing volume in the project
func (p *Project) UpdateVolume(volumeID uint, name string, capacity uint32) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Check for duplicate name (except for the volume being updated)
	for _, v := range p.volumes {
		if v.GetVolumeID() != volumeID && v.GetName() == name {
			return projecterrors.ErrDuplicateVolumeName
		}
	}

	// Find and update the volume
	for i := range p.volumes {
		if p.volumes[i].GetVolumeID() == volumeID {
			if err := p.volumes[i].Update(name, capacity); err != nil {
				return err
			}
			p.updateTimestamp()
			return nil
		}
	}

	return projecterrors.ErrVolumeNotFound
}

// GetVolumes returns a copy of project volumes
func (p *Project) GetVolumes() []Volume {
	volumes := make([]Volume, len(p.volumes))
	copy(volumes, p.volumes)
	return volumes
}

// GetVolumeByID returns a volume by its ID
func (p *Project) GetVolumeByID(volumeID uint) (*Volume, error) {
	for i := range p.volumes {
		if p.volumes[i].GetVolumeID() == volumeID {
			volume := p.volumes[i]
			return &volume, nil
		}
	}
	return nil, projecterrors.ErrVolumeNotFound
}

// GetVolumeByName returns a volume by its name
func (p *Project) GetVolumeByName(name string) (*Volume, error) {
	for i := range p.volumes {
		if p.volumes[i].GetName() == name {
			volume := p.volumes[i]
			return &volume, nil
		}
	}
	return nil, projecterrors.ErrVolumeNotFound
}

// setVolumeID sets the ID for a volume (for use by repository and tests only)
func (p *Project) setVolumeID(name string, volumeID uint) error {
	for i := range p.volumes {
		if p.volumes[i].GetName() == name {
			p.volumes[i].setVolumeID(volumeID)
			return nil
		}
	}
	return projecterrors.ErrVolumeNotFound
}

// SetFQDN sets the fully qualified domain name
func (p *Project) SetFQDN(fqdn string) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if fqdn == "" {
		p.fqdn = nil
	} else {
		p.fqdn = &fqdn
	}
	p.updateTimestamp()
	return nil
}

// UpdateName updates the project name
func (p *Project) UpdateName(name string) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if name == "" {
		return projecterrors.ErrNameRequired
	}

	p.name = name
	p.updateTimestamp()
	return nil
}

// UpdateSlug updates the project slug
func (p *Project) UpdateSlug(slug ProjectSlug) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if slug.IsEmpty() {
		return projecterrors.ErrSlugRequired
	}

	p.slug = slug
	p.updateTimestamp()
	return nil
}

// UpdatePlan updates the project plan
func (p *Project) UpdatePlan(plan string) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if plan == "" {
		p.plan = nil
	} else {
		p.plan = &plan
	}
	p.updateTimestamp()
	return nil
}

// UpdateStatus updates the project status
func (p *Project) UpdateStatus(status ProjectStatus) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if !status.IsValid() {
		return projecterrors.ErrInvalidProjectData
	}

	p.status = status
	p.updateTimestamp()
	return nil
}

// UpdateResourceLimits updates the resource limits
func (p *Project) UpdateResourceLimits(limits ResourceLimits) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Validate limits
	if err := limits.Validate(); err != nil {
		return err
	}

	p.limits = limits
	p.updateTimestamp()
	return nil
}

// Delete soft deletes the project
func (p *Project) Delete() error {
	if p.isDeleted {
		return nil // Already deleted
	}

	p.isDeleted = true
	now := time.Now()
	p.deletedAt = &now
	p.updatedAt = &now

	// Soft delete all users
	for i := range p.users {
		_ = p.users[i].SoftDelete() // Error ignored as SoftDelete always succeeds
	}

	return nil
}

// Restore restores a soft deleted project
func (p *Project) Restore() error {
	if !p.isDeleted {
		return nil // Not deleted
	}

	p.isDeleted = false
	p.deletedAt = nil
	p.updateTimestamp()

	// Note: Users are not automatically restored
	// This allows for selective user restoration

	return nil
}

// IsSoftDeleted returns whether the project is soft deleted
func (p *Project) IsSoftDeleted() bool {
	return p.isDeleted
}

// ValidateInvariants validates all domain invariants
func (p *Project) ValidateInvariants() error {
	// Invariant 1: Project must have at least one owner
	if !p.HasOwner() {
		return projecterrors.ErrOwnerRequired
	}

	// Invariant 2: Project name cannot be empty
	if p.name == "" {
		return projecterrors.ErrNameRequired
	}

	// Invariant 3: Project slug must be valid
	if p.slug.IsEmpty() {
		return projecterrors.ErrSlugRequired
	}

	// Invariant 4: Status must be valid
	if !p.status.IsValid() {
		return projecterrors.ErrInvalidProjectData
	}

	return nil
}

// Private helper methods

func (p *Project) hasOtherOwner(excludeUserID uint) bool {
	for _, user := range p.users {
		if user.GetUserID() != excludeUserID && user.IsOwner() && user.IsActive() {
			return true
		}
	}
	return false
}

func (p *Project) updateTimestamp() {
	now := time.Now()
	p.updatedAt = &now
}
