package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"go.uber.org/zap"
)

// ProjectService defines the interface for project-related business logic
type ProjectService interface {
	// CreateProject creates a new project with the given parameters
	// limits is required, fqdn and plan are optional
	// slug is automatically generated from name
	CreateProject(ctx context.Context, name string, ownerID uint, limits value.ResourceLimits, fqdn *string, plan *value.Plan) (*model.Project, error)

	// GetProject retrieves a project by ID
	GetProject(ctx context.Context, projectID uint) (*model.Project, error)

	// GetProjectBySlug retrieves a project by slug
	GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error)

	// UpdateProject updates an existing project
	// actingUserID is the user performing the update (for quota validation)
	UpdateProject(ctx context.Context, projectID uint, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error)

	// UpdateProjectBySlug updates an existing project by slug
	// actingUserID is the user performing the update (for quota validation)
	UpdateProjectBySlug(ctx context.Context, slug string, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error)

	// DeleteProject soft deletes a project
	DeleteProject(ctx context.Context, projectID uint) error

	// DeleteProjectBySlug soft deletes a project by slug
	DeleteProjectBySlug(ctx context.Context, slug string) error

	// ListProjects retrieves all projects for a user
	ListProjects(ctx context.Context, userID uint) ([]*model.Project, error)

	// CountProjectsByUserID returns the number of active projects for a user
	CountProjectsByUserID(ctx context.Context, userID uint) (int, error)

	// CheckProjectNameExists checks if a project name already exists for a user
	CheckProjectNameExists(ctx context.Context, name string, userID uint) (bool, error)
}

// projectService is the concrete implementation of ProjectService
type projectService struct {
	projectRepo       repository.ProjectRepository
	slugService       SlugService
	validationService ValidationService
	logger            logger.Logger
}

// NewProjectService creates a new instance of ProjectService
func NewProjectService(projectRepo repository.ProjectRepository, slugService SlugService, validationService ValidationService, log logger.Logger) ProjectService {
	return &projectService{
		projectRepo:       projectRepo,
		slugService:       slugService,
		validationService: validationService,
		logger:            log,
	}
}

// CreateProject creates a new project with validation
// slug is automatically generated from name
// limits is required, fqdn and plan are optional
func (s *projectService) CreateProject(ctx context.Context, name string, ownerID uint, limits value.ResourceLimits, fqdn *string, plan *value.Plan) (*model.Project, error) {
	s.logger.Info(ctx, "create project started",
		zap.String("name", name),
		zap.Uint("owner_id", ownerID),
	)

	// Determine the plan (default to Eco if not provided)
	selectedPlan := value.PlanEco
	if plan != nil {
		selectedPlan = *plan
	}

	// Validate Free plan resources (must match fixed values)
	if err := s.validationService.ValidateFreeResources(selectedPlan, limits); err != nil {
		s.logger.Error(ctx, "Free plan resources validation failed",
			zap.String("plan", selectedPlan.String()),
			zap.Error(err),
		)
		return nil, err
	}

	// Validate free tier limits (beta period restrictions)
	if err := s.validationService.ValidateFreeTierLimits(selectedPlan, limits); err != nil {
		s.logger.Error(ctx, "free tier limits validation failed",
			zap.String("plan", selectedPlan.String()),
			zap.Error(err),
		)
		return nil, err
	}

	// Validate Free plan project limit (1 per user)
	if err := s.validationService.ValidateFreePlanLimit(ctx, ownerID, selectedPlan); err != nil {
		s.logger.Error(ctx, "Free plan limit validation failed",
			zap.Uint("owner_id", ownerID),
			zap.String("plan", selectedPlan.String()),
			zap.Error(err),
		)
		return nil, err
	}

	// Check if project name already exists for this user
	exists, err := s.CheckProjectNameExists(ctx, name, ownerID)
	if err != nil {
		s.logger.Error(ctx, "failed to check project name exists",
			zap.String("name", name),
			zap.Uint("owner_id", ownerID),
			zap.Error(err),
		)
		return nil, err
	}
	if exists {
		s.logger.Error(ctx, "project name already exists",
			zap.String("name", name),
			zap.Uint("owner_id", ownerID),
		)
		return nil, projecterrors.ErrProjectNameExists
	}

	// Generate slug
	slug, err := s.slugService.GenerateSlug(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed to generate slug",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}

	// Create the project aggregate with all fields (including optional ones)
	// This ensures created_at and updated_at are the same at creation time
	// Use selectedPlan (which defaults to Eco if plan was nil)
	project, err := model.NewProject(name, slug, ownerID, limits, fqdn, &selectedPlan)
	if err != nil {
		s.logger.Error(ctx, "failed to create project model",
			zap.String("name", name),
			zap.String("slug", slug.String()),
			zap.Error(err),
		)
		return nil, err
	}

	// Persist the project
	if err := s.projectRepo.Create(ctx, project); err != nil {
		s.logger.Error(ctx, "failed to persist project",
			zap.String("name", name),
			zap.String("slug", slug.String()),
			zap.Error(err),
		)
		return nil, projecterrors.ErrProjectCreationFailed
	}

	s.logger.Info(ctx, "create project completed",
		zap.Uint("project_id", project.ProjectID()),
		zap.String("name", name),
		zap.String("slug", slug.String()),
	)
	return project, nil
}

// GetProject retrieves a project by ID
func (s *projectService) GetProject(ctx context.Context, projectID uint) (*model.Project, error) {
	s.logger.Info(ctx, "get project started",
		zap.Uint("project_id", projectID),
	)

	if projectID == 0 {
		s.logger.Error(ctx, "invalid project ID",
			zap.Uint("project_id", projectID),
		)
		return nil, projecterrors.ErrInvalidProjectID
	}

	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project by ID",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "get project completed",
		zap.Uint("project_id", projectID),
		zap.String("name", project.Name()),
	)
	return project, nil
}

// GetProjectBySlug retrieves a project by slug
func (s *projectService) GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error) {
	s.logger.Info(ctx, "get project by slug started",
		zap.String("slug", slug),
	)

	// Validate slug format before repository lookup
	// This returns ErrSlugInvalidFormat for malformed slugs (wrong length/prefix)
	// which maps to 400 Bad Request instead of 404 Not Found
	_, err := value.NewProjectSlug(slug)
	if err != nil {
		s.logger.Error(ctx, "invalid slug format",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	project, err := s.projectRepo.FindBySlug(ctx, slug)
	if err != nil {
		s.logger.Error(ctx, "failed to find project by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "get project by slug completed",
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
	)
	return project, nil
}

// UpdateProject updates an existing project
func (s *projectService) UpdateProject(ctx context.Context, projectID uint, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error) {
	s.logger.Info(ctx, "update project started",
		zap.Uint("project_id", projectID),
		zap.Uint("acting_user_id", actingUserID),
	)

	if projectID == 0 {
		s.logger.Error(ctx, "invalid project ID",
			zap.Uint("project_id", projectID),
		)
		return nil, projecterrors.ErrInvalidProjectID
	}

	// Retrieve the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project for update",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// Store original plan before update
	originalPlan, hadPlan := project.Plan()
	if !hadPlan {
		// Pre-existing projects without a plan default to Eco
		originalPlan = value.PlanEco
	}

	// Apply the update function
	if err := updateFn(project); err != nil {
		s.logger.Error(ctx, "failed to apply update function",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// Validate updated project state
	// Always run validations - treat NULL plan as default (Eco)
	plan, hasPlan := project.Plan()
	if !hasPlan {
		// Pre-existing projects without a plan default to Eco
		plan = value.PlanEco
	}

	limits := project.Limits()

	// Free plan must use fixed resources
	if err := s.validationService.ValidateFreeResources(plan, limits); err != nil {
		s.logger.Error(ctx, "free plan resource validation failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// Check beta tier limits (always enforced)
	if err := s.validationService.ValidateFreeTierLimits(plan, limits); err != nil {
		s.logger.Error(ctx, "beta tier limit validation failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	// If converting to Free plan (not already Free), check project count for the acting user
	// This prevents blocking updates to existing Free plan projects (e.g., renaming)
	if plan == value.PlanFree && originalPlan != value.PlanFree {
		if err := s.validationService.ValidateFreePlanLimit(ctx, actingUserID, plan); err != nil {
			s.logger.Error(ctx, "free plan limit validation failed",
				zap.Uint("project_id", projectID),
				zap.Uint("acting_user_id", actingUserID),
				zap.Error(err),
			)
			return nil, err
		}
	}

	// Save the updated project
	if err := s.projectRepo.Save(ctx, project); err != nil {
		s.logger.Error(ctx, "failed to save updated project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "update project completed",
		zap.Uint("project_id", projectID),
		zap.String("name", project.Name()),
	)
	return project, nil
}

// UpdateProjectBySlug updates an existing project by slug
func (s *projectService) UpdateProjectBySlug(ctx context.Context, slug string, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error) {
	s.logger.Info(ctx, "update project by slug started",
		zap.String("slug", slug),
		zap.Uint("acting_user_id", actingUserID),
	)

	if slug == "" {
		s.logger.Error(ctx, "invalid slug (empty)",
			zap.String("slug", slug),
		)
		return nil, projecterrors.ErrInvalidSlug
	}

	// Retrieve the project
	project, err := s.projectRepo.FindBySlug(ctx, slug)
	if err != nil {
		s.logger.Error(ctx, "failed to find project by slug for update",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	// Store original plan before update
	originalPlan, hadPlan := project.Plan()
	if !hadPlan {
		// Pre-existing projects without a plan default to Eco
		originalPlan = value.PlanEco
	}

	// Apply the update function
	if err := updateFn(project); err != nil {
		s.logger.Error(ctx, "failed to apply update function",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	// Validate updated project state
	// Always run validations - treat NULL plan as default (Eco)
	plan, hasPlan := project.Plan()
	if !hasPlan {
		// Pre-existing projects without a plan default to Eco
		plan = value.PlanEco
	}

	limits := project.Limits()

	// Free plan must use fixed resources
	if err := s.validationService.ValidateFreeResources(plan, limits); err != nil {
		s.logger.Error(ctx, "free plan resource validation failed",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	// Check beta tier limits (always enforced)
	if err := s.validationService.ValidateFreeTierLimits(plan, limits); err != nil {
		s.logger.Error(ctx, "beta tier limit validation failed",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	// If converting to Free plan (not already Free), check project count for the acting user
	// This prevents blocking updates to existing Free plan projects (e.g., renaming)
	if plan == value.PlanFree && originalPlan != value.PlanFree {
		if err := s.validationService.ValidateFreePlanLimit(ctx, actingUserID, plan); err != nil {
			s.logger.Error(ctx, "free plan limit validation failed",
				zap.String("slug", slug),
				zap.Uint("acting_user_id", actingUserID),
				zap.Error(err),
			)
			return nil, err
		}
	}

	// Save the updated project
	if err := s.projectRepo.Save(ctx, project); err != nil {
		s.logger.Error(ctx, "failed to save updated project",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "update project by slug completed",
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
	)
	return project, nil
}

// DeleteProject soft deletes a project
func (s *projectService) DeleteProject(ctx context.Context, projectID uint) error {
	s.logger.Info(ctx, "delete project started",
		zap.Uint("project_id", projectID),
	)

	if projectID == 0 {
		s.logger.Error(ctx, "invalid project ID",
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrInvalidProjectID
	}

	// Retrieve the project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project for deletion",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	// Soft delete the project (also deletes all active users)
	if err := project.SoftDelete(); err != nil {
		s.logger.Error(ctx, "failed to soft delete project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	// Save the deleted state
	if err := s.projectRepo.Save(ctx, project); err != nil {
		s.logger.Error(ctx, "failed to save deleted project state",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "delete project completed",
		zap.Uint("project_id", projectID),
	)
	return nil
}

// DeleteProjectBySlug soft deletes a project by slug
func (s *projectService) DeleteProjectBySlug(ctx context.Context, slug string) error {
	s.logger.Info(ctx, "delete project by slug started",
		zap.String("slug", slug),
	)

	if slug == "" {
		s.logger.Error(ctx, "invalid slug (empty)",
			zap.String("slug", slug),
		)
		return projecterrors.ErrInvalidSlug
	}

	// Retrieve the project
	project, err := s.projectRepo.FindBySlug(ctx, slug)
	if err != nil {
		s.logger.Error(ctx, "failed to find project by slug for deletion",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return err
	}

	// Soft delete the project (also deletes all active users)
	if err := project.SoftDelete(); err != nil {
		s.logger.Error(ctx, "failed to soft delete project",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return err
	}

	// Save the deleted state
	if err := s.projectRepo.Save(ctx, project); err != nil {
		s.logger.Error(ctx, "failed to save deleted project state",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "delete project by slug completed",
		zap.String("slug", slug),
	)
	return nil
}

// ListProjects retrieves all projects for a user
func (s *projectService) ListProjects(ctx context.Context, userID uint) ([]*model.Project, error) {
	s.logger.Info(ctx, "list projects started",
		zap.Uint("user_id", userID),
	)

	if userID == 0 {
		s.logger.Error(ctx, "invalid user ID",
			zap.Uint("user_id", userID),
		)
		return nil, projecterrors.ErrInvalidUserID
	}

	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to find projects by user ID",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info(ctx, "list projects completed",
		zap.Uint("user_id", userID),
		zap.Int("count", len(projects)),
	)
	return projects, nil
}

// CountProjectsByUserID returns the number of active projects for a user
func (s *projectService) CountProjectsByUserID(ctx context.Context, userID uint) (int, error) {
	s.logger.Info(ctx, "count projects by user ID started",
		zap.Uint("user_id", userID),
	)

	if userID == 0 {
		s.logger.Error(ctx, "invalid user ID",
			zap.Uint("user_id", userID),
		)
		return 0, projecterrors.ErrInvalidUserID
	}

	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to find projects by user ID",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return 0, err
	}

	// Count only active (non-deleted) projects
	count := 0
	for _, project := range projects {
		if !project.IsDeleted() {
			count++
		}
	}

	s.logger.Info(ctx, "count projects by user ID completed",
		zap.Uint("user_id", userID),
		zap.Int("count", count),
	)
	return count, nil
}

// CheckProjectNameExists checks if a project name already exists for a user
func (s *projectService) CheckProjectNameExists(ctx context.Context, name string, userID uint) (bool, error) {
	s.logger.Info(ctx, "check project name exists started",
		zap.String("name", name),
		zap.Uint("user_id", userID),
	)

	if name == "" {
		s.logger.Error(ctx, "project name is required",
			zap.String("name", name),
		)
		return false, projecterrors.ErrNameRequired
	}
	if userID == 0 {
		s.logger.Error(ctx, "invalid user ID",
			zap.Uint("user_id", userID),
		)
		return false, projecterrors.ErrInvalidUserID
	}

	exists, err := s.projectRepo.ExistsByNameAndUserID(ctx, name, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to check project name exists",
			zap.String("name", name),
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		return false, err
	}

	s.logger.Info(ctx, "check project name exists completed",
		zap.String("name", name),
		zap.Uint("user_id", userID),
		zap.Bool("exists", exists),
	)
	return exists, nil
}
