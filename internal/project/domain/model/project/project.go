package model

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// Project represents a project aggregate root
// It manages ProjectUser entities and enforces domain invariants
type Project struct {
	projectID          uint
	name               string
	slug               value.ProjectSlug
	fqdn               *string
	status             value.ProjectStatus
	operationStatus    value.ProjectOperationStatus
	activeDeploymentID *uint // ID of the deployment that currently owns the deploying status
	plan               *string
	limits             value.ResourceLimits
	users              []ProjectUser // Aggregate's internal entities
	isDeleted          bool
	deletedAt          *time.Time
	createdAt          time.Time
	updatedAt          time.Time
}

// NewProject creates a new project with an initial owner
// fqdn and plan are optional parameters (pass nil if not needed)
func NewProject(name string, slug value.ProjectSlug, ownerID uint, limits value.ResourceLimits, fqdn *string, plan *string) (*Project, error) {
	if name == "" {
		return nil, projecterrors.ErrNameRequired
	}
	if ownerID == 0 {
		return nil, projecterrors.ErrOwnerIDRequired
	}

	now := time.Now()
	project := &Project{
		name:            name,
		slug:            slug,
		status:          value.ProjectStatusActive,
		operationStatus: value.ProjectOperationStatusNothing,
		limits:          limits,
		users:           make([]ProjectUser, 0),
		isDeleted:       false,
		createdAt:       now,
		updatedAt:       now,
	}

	// Set optional fields by copying the pointer values
	// This ensures created_at and updated_at remain the same
	if fqdn != nil {
		fqdnCopy := *fqdn
		project.fqdn = &fqdnCopy
	}

	if plan != nil {
		planCopy := *plan
		project.plan = &planCopy
	}

	// Add the initial owner without projectID (will be set later)
	// Create ProjectUser directly to avoid validation error
	owner := &ProjectUser{
		projectID: 0, // Will be set when project ID is assigned
		userID:    ownerID,
		role:      value.ProjectUserRoleOwner,
		isDeleted: false,
		createdAt: now,
		updatedAt: now,
	}
	project.users = append(project.users, *owner)

	return project, nil
}

// ReconstructProject reconstructs a project from persistence without initial owner
// This is used when loading a project from the database
func ReconstructProject(
	projectID uint,
	name string,
	slug value.ProjectSlug,
	status value.ProjectStatus,
	operationStatus value.ProjectOperationStatus,
	activeDeploymentID *uint,
	limits value.ResourceLimits,
	createdAt time.Time,
	updatedAt time.Time,
	isDeleted bool,
	deletedAt *time.Time,
) *Project {
	return &Project{
		projectID:          projectID,
		name:               name,
		slug:               slug,
		status:             status,
		operationStatus:    operationStatus,
		activeDeploymentID: activeDeploymentID,
		limits:             limits,
		users:              make([]ProjectUser, 0), // Will be loaded separately
		isDeleted:          isDeleted,
		deletedAt:          deletedAt,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
	}
}

// ProjectID returns the project ID
func (p *Project) ProjectID() uint {
	return p.projectID
}

// Name returns the project name
func (p *Project) Name() string {
	return p.name
}

// Slug returns the project slug
func (p *Project) Slug() value.ProjectSlug {
	return p.slug
}

// FQDN returns the fully qualified domain name and whether it is set
func (p *Project) FQDN() (string, bool) {
	if p.fqdn == nil {
		return "", false
	}
	return *p.fqdn, true
}

// Status returns the project status
func (p *Project) Status() value.ProjectStatus {
	return p.status
}

// OperationStatus returns the project operation status
func (p *Project) OperationStatus() value.ProjectOperationStatus {
	return p.operationStatus
}

// ActiveDeploymentID returns the ID of the deployment that currently owns the deploying status
// Returns (0, false) if no active deployment
func (p *Project) ActiveDeploymentID() (uint, bool) {
	if p.activeDeploymentID == nil {
		return 0, false
	}
	return *p.activeDeploymentID, true
}

// Plan returns the project plan and whether it is set
func (p *Project) Plan() (string, bool) {
	if p.plan == nil {
		return "", false
	}
	return *p.plan, true
}

// Limits returns the resource limits
func (p *Project) Limits() value.ResourceLimits {
	return p.limits
}

// Users returns a copy of project users
func (p *Project) Users() []ProjectUser {
	users := make([]ProjectUser, len(p.users))
	copy(users, p.users)
	return users
}

// CreatedAt returns the creation time
func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update time
func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}

// IsDeleted returns whether the project is soft deleted
func (p *Project) IsDeleted() bool {
	return p.isDeleted
}

// DeletedAt returns the deletion time
func (p *Project) DeletedAt() (time.Time, bool) {
	if p.deletedAt == nil {
		return time.Time{}, false
	}
	return *p.deletedAt, true
}

// SetProjectID sets the project ID (typically set by repository after persistence)
func (p *Project) SetProjectID(id uint) {
	p.projectID = id
	// Update projectID for all users
	for i := range p.users {
		p.users[i].projectID = id
	}
}

// SetName updates the project name
func (p *Project) SetName(name string) error {
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

// SetSlug updates the project slug
func (p *Project) SetSlug(slug value.ProjectSlug) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	p.slug = slug
	p.updateTimestamp()
	return nil
}

// SetFQDN sets the fully qualified domain name
// If fqdn is empty, it clears the FQDN
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

// SetPlan updates the project plan
// If plan is empty, it clears the plan
func (p *Project) SetPlan(plan string) error {
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

// SetStatus updates the project status
func (p *Project) SetStatus(status value.ProjectStatus) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	p.status = status
	p.updateTimestamp()
	return nil
}

// SetResourceLimits updates the resource limits
func (p *Project) SetResourceLimits(limits value.ResourceLimits) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	p.limits = limits
	p.updateTimestamp()
	return nil
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

// GetUserByID returns a user by their ID
func (p *Project) GetUserByID(userID uint) (*ProjectUser, error) {
	for i := range p.users {
		if p.users[i].UserID() == userID && p.users[i].IsActive() {
			user := p.users[i]
			return &user, nil
		}
	}
	return nil, projecterrors.ErrUserNotInProject
}

// HasUser checks if a user is in the project
func (p *Project) HasUser(userID uint) bool {
	for _, user := range p.users {
		if user.UserID() == userID && user.IsActive() {
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

// AddUser adds a new user to the project
func (p *Project) AddUser(userID uint, role value.ProjectUserRole) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Find user if exists
	for i := range p.users {
		if p.users[i].UserID() == userID {
			if p.users[i].IsActive() {
				return projecterrors.ErrUserAlreadyInProject
			}
			// Deleted user found - restore it
			if err := p.users[i].Restore(); err != nil {
				return err
			}
			// Change role to new role
			if err := p.users[i].ChangeRole(role); err != nil {
				return err
			}
			p.updateTimestamp()
			return nil
		}
	}

	// No user found - create new
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
		if p.users[i].UserID() == userID && p.users[i].IsActive() {
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
	if err := p.users[userIndex].SoftDelete(); err != nil {
		return err
	}
	p.updateTimestamp()

	return nil
}

// ChangeUserRole changes a user's role in the project
func (p *Project) ChangeUserRole(userID uint, newRole value.ProjectUserRole) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Find the user
	userIndex := -1
	for i := range p.users {
		if p.users[i].UserID() == userID && p.users[i].IsActive() {
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

// Delete soft deletes the project
func (p *Project) SoftDelete() error {
	if p.isDeleted {
		return nil // Already deleted
	}

	p.isDeleted = true
	now := time.Now()
	p.deletedAt = &now
	p.updatedAt = now

	// Soft delete all users
	for i := range p.users {
		if p.users[i].IsActive() {
			if err := p.users[i].SoftDelete(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Private helper methods

func (p *Project) hasOtherOwner(excludeUserID uint) bool {
	for _, user := range p.users {
		if user.UserID() != excludeUserID && user.IsOwner() && user.IsActive() {
			return true
		}
	}
	return false
}

func (p *Project) updateTimestamp() {
	now := time.Now()
	p.updatedAt = now
}

// StartDeploy transitions the project to deploying status and records which deployment owns the lock
// Returns error if project is already in an active operation
func (p *Project) StartDeploy(deploymentID uint) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	if p.operationStatus != value.ProjectOperationStatusNothing {
		return projecterrors.ErrInvalidStatusTransition
	}

	p.operationStatus = value.ProjectOperationStatusDeploying
	p.activeDeploymentID = &deploymentID
	p.updateTimestamp()
	return nil
}

// CompleteDeploy resets the operation status to nothing and clears active deployment ID
// This is called when a deployment completes (success, failure, or cancellation)
// Returns an error if the deployment ID does not own the lock or if the project is deleted
func (p *Project) CompleteDeploy(deploymentID uint) error {
	if p.isDeleted {
		return projecterrors.ErrCannotModifyDeletedProject
	}

	// Verify that this deployment owns the lock
	if p.activeDeploymentID == nil || *p.activeDeploymentID != deploymentID {
		return projecterrors.ErrInvalidStatusTransition
	}

	p.operationStatus = value.ProjectOperationStatusNothing
	p.activeDeploymentID = nil
	p.updateTimestamp()
	return nil
}
