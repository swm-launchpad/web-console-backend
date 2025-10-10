package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gosimple/slug"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// SlugService defines the interface for slug-related operations
type SlugService interface {
	// EnsureUniqueSlug validates that a slug is unique in the system
	EnsureUniqueSlug(ctx context.Context, slug value.ProjectSlug) error

	// GenerateSlugFromName generates a slug from project name with timestamp and random suffix
	// Format: {baseSlug}-{timestamp}-{random}
	// Uses gosimple/slug for Unicode transliteration (supports Korean)
	GenerateSlugFromName(ctx context.Context, name string) (value.ProjectSlug, error)
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

// GenerateSlugFromName generates a slug from project name with timestamp and random suffix
// Format: {baseSlug}-{timestamp}-{random}
// No retry logic - returns error immediately if slug already exists
func (s *slugService) GenerateSlugFromName(ctx context.Context, name string) (value.ProjectSlug, error) {
	// Generate slug with timestamp and random suffix
	generatedSlug := s.generateSlugFromName(name)

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

// generateSlugFromName generates a slug from project name using gosimple/slug library
// Format: {baseSlug}-{timestamp}-{random}
// - baseSlug: transliterated from name using gosimple/slug (supports Korean)
// - timestamp: YYYYMMDDHHMMSS (14 digits)
// - random: 4 digits (1000-9999)
// - Max length: 63 characters (subdomain constraint)
func (s *slugService) generateSlugFromName(name string) string {
	// Use the slug library to handle Unicode transliteration (including Korean)
	baseSlug := slug.Make(name)

	// If the slug is empty, use "project" as fallback
	if baseSlug == "" {
		baseSlug = "project"
	}

	// Generate timestamp (14 digits: YYYYMMDDHHMMSS)
	timestamp := time.Now().Format("20060102150405")

	// Generate random 4-digit suffix
	randomSuffix := s.generateRandomSuffix()

	// Combine: baseSlug-timestamp-random
	generatedSlug := fmt.Sprintf("%s-%s-%s", baseSlug, timestamp, randomSuffix)

	// Limit length to 63 characters (subdomain constraint)
	if len(generatedSlug) > 63 {
		// timestamp(14) + random(4) + hyphens(2) = 20 characters needed
		maxBaseLength := 63 - 20
		if maxBaseLength > 0 && len(baseSlug) > maxBaseLength {
			baseSlug = baseSlug[:maxBaseLength]
			// Remove trailing hyphen if exists
			for len(baseSlug) > 0 && baseSlug[len(baseSlug)-1] == '-' {
				baseSlug = baseSlug[:len(baseSlug)-1]
			}
			generatedSlug = fmt.Sprintf("%s-%s-%s", baseSlug, timestamp, randomSuffix)
		}
	}

	return generatedSlug
}

// generateRandomSuffix generates a random suffix for slug uniqueness
func (s *slugService) generateRandomSuffix() string {
	// Create a new random source with current time as seed
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	// Generate 4-digit random number (1000-9999)
	randomNum := r.Intn(9000) + 1000
	return fmt.Sprintf("%d", randomNum)
}
