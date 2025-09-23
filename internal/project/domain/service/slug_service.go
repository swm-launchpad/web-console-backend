package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

// SlugService defines the interface for slug-related operations
type SlugService interface {
	// EnsureUniqueSlug validates that a slug is unique in the system
	EnsureUniqueSlug(ctx context.Context, slug model.ProjectSlug) error
}

// slugService is the concrete implementation of SlugService
type slugService struct {
	projectRepo repository.ProjectRepository
}

// NewSlugService creates a new instance of SlugService
func NewSlugService(projectRepo repository.ProjectRepository) SlugService {
	return &slugService{
		projectRepo: projectRepo,
	}
}

// EnsureUniqueSlug ensures that a slug is unique
func (s *slugService) EnsureUniqueSlug(ctx context.Context, slug model.ProjectSlug) error {
	exists, err := s.projectRepo.ExistsBySlug(ctx, slug.String())
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	if exists {
		return projecterrors.ErrSlugAlreadyExists
	}

	return nil
}
