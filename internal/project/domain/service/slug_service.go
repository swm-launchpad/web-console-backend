package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// SlugService defines the interface for slug-related operations
type SlugService interface {
	// EnsureUniqueSlug validates that a slug is unique in the system
	EnsureUniqueSlug(ctx context.Context, slug value.ProjectSlug) error

	// GenerateSlug generates a unique slug for a project
	// Format: p{timestamp}{random} (23 characters fixed)
	// Example: p2025011812000012345678
	GenerateSlug(ctx context.Context) (value.ProjectSlug, error)
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
func (s *slugService) EnsureUniqueSlug(ctx context.Context, slug value.ProjectSlug) error {
	exists, err := s.projectRepo.ExistsBySlug(ctx, slug.String())
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	if exists {
		return projecterrors.ErrSlugAlreadyExists
	}

	return nil
}

// GenerateSlug generates a unique slug for a project
// Format: p{timestamp}{random} (23 characters fixed)
// Example: p2025011812000012345678
// No retry logic - returns error immediately if slug already exists
func (s *slugService) GenerateSlug(ctx context.Context) (value.ProjectSlug, error) {
	// Generate slug with timestamp and random suffix
	generatedSlug := s.generateSlug()

	// Validate slug format
	slugModel, err := value.NewProjectSlug(generatedSlug)
	if err != nil {
		return value.ProjectSlug{}, err
	}

	// Check if slug is unique
	err = s.EnsureUniqueSlug(ctx, *slugModel)
	if err != nil {
		return value.ProjectSlug{}, err
	}

	return *slugModel, nil
}

// generateSlug generates a slug in the format: p{timestamp}{random}
// - p: prefix (1 character)
// - timestamp: YYYYMMDDHHMMSS (14 digits)
// - random: lowercase alphanumeric (8 characters)
// - Total: 23 characters (fixed)
func (s *slugService) generateSlug() string {
	// Generate timestamp (14 digits: YYYYMMDDHHMMSS)
	timestamp := time.Now().Format("20060102150405")

	// Generate random 8-character suffix (lowercase alphanumeric)
	randomSuffix := s.generateRandomSuffix()

	// Combine: p + timestamp + random
	return fmt.Sprintf("p%s%s", timestamp, randomSuffix)
}

// generateRandomSuffix generates an 8-character lowercase alphanumeric random suffix
func (s *slugService) generateRandomSuffix() string {
	// Create a new random source with current time as seed
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[r.Intn(len(chars))]
	}

	return string(result)
}
