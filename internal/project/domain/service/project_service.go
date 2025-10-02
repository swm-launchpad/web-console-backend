package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

// ProjectService defines the interface for project-related business logic
type ProjectService interface {
	// CreateProject creates a new project with the given parameters
	// limits is required, fqdn and plan are optional
	// slug is automatically generated from name
	CreateProject(ctx context.Context, name string, ownerID uint, limits value.ResourceLimits, fqdn *string, plan *string) (*model.Project, error)

	// GetProject retrieves a project by ID
	GetProject(ctx context.Context, projectID uint) (*model.Project, error)

	// UpdateProject updates an existing project
	UpdateProject(ctx context.Context, projectID uint, updateFn func(*model.Project) error) (*model.Project, error)

	// DeleteProject soft deletes a project
	DeleteProject(ctx context.Context, projectID uint) error

	// ListProjects retrieves all projects for a user
	ListProjects(ctx context.Context, userID uint) ([]*model.Project, error)

	// CountProjectsByUserID returns the number of active projects for a user
	CountProjectsByUserID(ctx context.Context, userID uint) (int, error)

	// CheckProjectNameExists checks if a project name already exists for a user
	CheckProjectNameExists(ctx context.Context, name string, userID uint) (bool, error)
}

// projectService is the concrete implementation of ProjectService
type projectService struct {
	projectRepo repository.ProjectRepository
	slugService SlugService
}

// NewProjectService creates a new instance of ProjectService
func NewProjectService(projectRepo repository.ProjectRepository, slugService SlugService) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		slugService: slugService,
	}
}

// CreateProject creates a new project with validation
// slug is automatically generated from name
// limits is required, fqdn and plan are optional
func (s *projectService) CreateProject(ctx context.Context, name string, ownerID uint, limits value.ResourceLimits, fqdn *string, plan *string) (*model.Project, error) {
	// Check if project name already exists for this user
	exists, err := s.CheckProjectNameExists(ctx, name, ownerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, projecterrors.ErrProjectNameExists
	}

	// Generate slug from name
	slug, err := s.slugService.GenerateSlugFromName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Create the project aggregate with all fields (including optional ones)
	// This ensures created_at and updated_at are the same at creation time
	project, err := model.NewProject(name, slug, ownerID, limits, fqdn, plan)
	if err != nil {
		return nil, err
	}

	// Persist the project
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, projecterrors.ErrProjectCreationFailed
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *projectService) GetProject(ctx context.Context, projectID uint) (*model.Project, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}

	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return project, nil
}

// UpdateProject updates an existing project
func (s *projectService) UpdateProject(ctx context.Context, projectID uint, updateFn func(*model.Project) error) (*model.Project, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}

	// Retrieve the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Apply the update function
	if err := updateFn(project); err != nil {
		return nil, err
	}

	// Save the updated project
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// DeleteProject soft deletes a project
func (s *projectService) DeleteProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return projecterrors.ErrInvalidProjectID
	}

	// Retrieve the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Soft delete the project (also deletes all active users)
	if err := project.SoftDelete(); err != nil {
		return err
	}

	// Save the deleted state
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return err
	}

	return nil
}

// ListProjects retrieves all projects for a user
func (s *projectService) ListProjects(ctx context.Context, userID uint) ([]*model.Project, error) {
	if userID == 0 {
		return nil, projecterrors.ErrInvalidUserID
	}

	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// CountProjectsByUserID returns the number of active projects for a user
func (s *projectService) CountProjectsByUserID(ctx context.Context, userID uint) (int, error) {
	if userID == 0 {
		return 0, projecterrors.ErrInvalidUserID
	}

	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Count only active (non-deleted) projects
	count := 0
	for _, project := range projects {
		if !project.IsDeleted() {
			count++
		}
	}

	return count, nil
}

// CheckProjectNameExists checks if a project name already exists for a user
func (s *projectService) CheckProjectNameExists(ctx context.Context, name string, userID uint) (bool, error) {
	if name == "" {
		return false, projecterrors.ErrNameRequired
	}
	if userID == 0 {
		return false, projecterrors.ErrInvalidUserID
	}

	exists, err := s.projectRepo.ExistsByNameAndUserID(ctx, name, userID)
	if err != nil {
		return false, err
	}

	return exists, nil
}
