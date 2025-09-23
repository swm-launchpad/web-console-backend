package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

// ProjectService defines the interface for project-related business logic
type ProjectService interface {
	// CreateProject creates a new project with the given parameters
	CreateProject(ctx context.Context, name string, slug model.ProjectSlug, ownerID uint) (*model.Project, error)

	// GetProject retrieves a project by ID
	GetProject(ctx context.Context, projectID uint) (*model.Project, error)

	// GetProjectBySlug retrieves a project by slug
	GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error)

	// UpdateProject updates an existing project
	UpdateProject(ctx context.Context, projectID uint, updateFn func(*model.Project) error) (*model.Project, error)

	// DeleteProject soft deletes a project
	DeleteProject(ctx context.Context, projectID uint) error

	// ListProjects retrieves all projects for a user
	ListProjects(ctx context.Context, userID uint) ([]*model.Project, error)

	// ListAllProjects retrieves all projects with pagination
	ListAllProjects(ctx context.Context, offset, limit int) ([]*model.Project, error)
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
func (s *projectService) CreateProject(ctx context.Context, name string, slug model.ProjectSlug, ownerID uint) (*model.Project, error) {
	// Validate slug uniqueness
	if err := s.slugService.EnsureUniqueSlug(ctx, slug); err != nil {
		return nil, err
	}

	// Create the project aggregate
	project, err := model.NewProject(name, slug, ownerID)
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

// GetProjectBySlug retrieves a project by slug
func (s *projectService) GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error) {
	if slug == "" {
		return nil, projecterrors.ErrSlugRequired
	}

	project, err := s.projectRepo.FindBySlug(ctx, slug)
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

	// Soft delete the project
	if err := project.Delete(); err != nil {
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

// ListAllProjects retrieves all projects with pagination
func (s *projectService) ListAllProjects(ctx context.Context, offset, limit int) ([]*model.Project, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	projects, err := s.projectRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	return projects, nil
}
