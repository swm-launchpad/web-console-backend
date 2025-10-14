package repository

import (
	"context"

	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
)

// ProjectRepository defines the interface for project data persistence
// This follows the repository pattern and is part of the domain layer
type ProjectRepository interface {
	// Create persists a new project and assigns its ID
	// The entire aggregate (including ProjectUsers and Volumes) should be saved
	Create(ctx context.Context, project *model.Project) error

	// Save updates an existing project
	// The entire aggregate (including ProjectUsers) should be saved
	Save(ctx context.Context, project *model.Project) error

	// FindByID retrieves a project by its ID
	// Should return the complete aggregate with all ProjectUsers
	FindByID(ctx context.Context, projectID uint) (*model.Project, error)

	// FindByIDForUpdate retrieves a project by its ID with row lock (SELECT FOR UPDATE)
	// Used for preventing race conditions in concurrent modifications
	FindByIDForUpdate(ctx context.Context, projectID uint) (*model.Project, error)

	// FindByUserID retrieves all projects for a specific user
	// Returns only projects where the user is an active member
	FindByUserID(ctx context.Context, userID uint) ([]*model.Project, error)

	// ExistsBySlug checks if a project with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// ExistsByNameAndUserID checks if a project with the given name exists for the user
	ExistsByNameAndUserID(ctx context.Context, name string, userID uint) (bool, error)

	// Delete soft deletes a project (sets is_deleted = true)
	Delete(ctx context.Context, projectID uint) error

	// FindProjectsWithActiveOperations finds all projects that have ongoing operations
	// Used for monitoring and preventing concurrent operations
	// Returns projects with operation_status != 'nothing'
	FindProjectsWithActiveOperations(ctx context.Context) ([]*model.Project, error)
}
