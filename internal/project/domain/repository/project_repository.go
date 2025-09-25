package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
)

// ProjectRepository defines the interface for project data persistence
// This follows the repository pattern and is part of the domain layer
type ProjectRepository interface {
	// Create persists a new project and assigns its ID
	// The entire aggregate (including ProjectUsers and Volumes) should be saved
	Create(ctx context.Context, project *model.Project) error

	// Save updates an existing project
	// The entire aggregate (including ProjectUsers and Volumes) should be saved
	Save(ctx context.Context, project *model.Project) error

	// FindByID retrieves a project by its ID
	// Should return the complete aggregate with all ProjectUsers and Volumes
	FindByID(ctx context.Context, projectID uint) (*model.Project, error)

	// FindBySlug retrieves a project by its slug
	// Should return the complete aggregate with all ProjectUsers and Volumes
	FindBySlug(ctx context.Context, slug string) (*model.Project, error)

	// FindByUserID retrieves all projects for a specific user
	// Returns only projects where the user is an active member
	FindByUserID(ctx context.Context, userID uint) ([]*model.Project, error)

	// ExistsBySlug checks if a project with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// Delete soft deletes a project (sets is_deleted = true)
	Delete(ctx context.Context, projectID uint) error

	// List retrieves projects with pagination
	// Returns only non-deleted projects
	List(ctx context.Context, offset, limit int) ([]*model.Project, error)

	// Count returns total number of non-deleted projects
	Count(ctx context.Context) (int64, error)
}
