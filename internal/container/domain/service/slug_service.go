package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// SlugService defines the interface for slug-related operations
type SlugService interface {
	// EnsureUniqueSlug validates that a slug is unique in the system
	EnsureUniqueSlug(ctx context.Context, slug value.ContainerSlug) error

	// GenerateSlug generates a unique slug for a container
	// Format: c{timestamp}{random} (23 characters fixed)
	// Example: c2025011812000012345678
	GenerateSlug(ctx context.Context) (value.ContainerSlug, error)
}

// slugService is the concrete implementation of SlugService
type slugService struct {
	containerRepo repository.ContainerRepository
}

// NewSlugService creates a new instance of SlugService
func NewSlugService(containerRepo repository.ContainerRepository) SlugService {
	return &slugService{
		containerRepo: containerRepo,
	}
}

// EnsureUniqueSlug ensures that a slug is unique across all containers
func (s *slugService) EnsureUniqueSlug(ctx context.Context, slug value.ContainerSlug) error {
	exists, err := s.containerRepo.ExistsBySlug(ctx, slug.String())
	if err != nil {
		return containererrors.ErrDatabaseOperation
	}

	if exists {
		return containererrors.ErrSlugAlreadyExists
	}

	return nil
}

// GenerateSlug generates a unique slug for a container
// Format: c{timestamp}{random} (23 characters fixed)
// Example: c2025011812000012345678
// No retry logic - returns error immediately if slug already exists
func (s *slugService) GenerateSlug(ctx context.Context) (value.ContainerSlug, error) {
	// Generate slug with timestamp and random suffix
	generatedSlug := s.generateSlug()

	// Validate slug format
	slugModel, err := value.NewContainerSlug(generatedSlug)
	if err != nil {
		return value.ContainerSlug{}, err
	}

	// Check if slug is unique
	err = s.EnsureUniqueSlug(ctx, slugModel)
	if err != nil {
		return value.ContainerSlug{}, err
	}

	return slugModel, nil
}

// generateSlug generates a slug in the format: c{timestamp}{random}
// - c: prefix (1 character)
// - timestamp: YYYYMMDDHHMMSS (14 digits)
// - random: lowercase alphanumeric (8 characters)
// - Total: 23 characters (fixed)
func (s *slugService) generateSlug() string {
	// Generate timestamp (14 digits: YYYYMMDDHHMMSS)
	timestamp := time.Now().Format("20060102150405")

	// Generate random 8-character suffix (lowercase alphanumeric)
	randomSuffix := s.generateRandomSuffix()

	// Combine: c + timestamp + random
	return fmt.Sprintf("c%s%s", timestamp, randomSuffix)
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
