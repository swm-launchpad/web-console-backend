package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
)

// VolumeSlugService defines the interface for volume slug-related operations
type VolumeSlugService interface {
	// EnsureUniqueSlug validates that a slug is unique in the system
	EnsureUniqueSlug(ctx context.Context, slug value.VolumeSlug) error

	// GenerateSlug generates a unique slug for a volume
	// Format: v{timestamp}{random} (23 characters fixed)
	// Example: v2025011812000012345678
	GenerateSlug(ctx context.Context) (value.VolumeSlug, error)
}

// volumeSlugService is the concrete implementation of VolumeSlugService
type volumeSlugService struct {
	volumeRepo repository.VolumeRepository
	logger     logger.Logger
}

// NewVolumeSlugService creates a new instance of VolumeSlugService
func NewVolumeSlugService(volumeRepo repository.VolumeRepository, log logger.Logger) VolumeSlugService {
	return &volumeSlugService{
		volumeRepo: volumeRepo,
		logger:     log,
	}
}

// EnsureUniqueSlug ensures that a slug is unique across all volumes
func (s *volumeSlugService) EnsureUniqueSlug(ctx context.Context, slug value.VolumeSlug) error {
	exists, err := s.volumeRepo.ExistsBySlug(ctx, slug.String())
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	if exists {
		s.logger.Warn(ctx, "Volume slug collision detected",
			zap.String("slug", slug.String()),
		)
		return projecterrors.ErrSlugAlreadyExists
	}

	return nil
}

// GenerateSlug generates a unique slug for a volume
// Format: v{timestamp}{random} (23 characters fixed)
// Example: v2025011812000012345678
// No retry logic - returns error immediately if slug already exists
func (s *volumeSlugService) GenerateSlug(ctx context.Context) (value.VolumeSlug, error) {
	// Generate slug with timestamp and random suffix
	generatedSlug := s.generateSlug()

	// Validate slug format
	slugModel, err := value.NewVolumeSlug(generatedSlug)
	if err != nil {
		return value.VolumeSlug{}, err
	}

	// Check if slug is unique
	err = s.EnsureUniqueSlug(ctx, slugModel)
	if err != nil {
		return value.VolumeSlug{}, err
	}

	return slugModel, nil
}

// generateSlug generates a slug in the format: v{timestamp}{random}
// - v: prefix (1 character)
// - timestamp: YYYYMMDDHHMMSS (14 digits)
// - random: lowercase alphanumeric (8 characters)
// - Total: 23 characters (fixed)
func (s *volumeSlugService) generateSlug() string {
	// Generate timestamp (14 digits: YYYYMMDDHHMMSS)
	timestamp := time.Now().Format("20060102150405")

	// Generate random 8-character suffix (lowercase alphanumeric)
	randomSuffix := s.generateRandomSuffix()

	// Combine: v + timestamp + random
	return fmt.Sprintf("v%s%s", timestamp, randomSuffix)
}

// generateRandomSuffix generates an 8-character lowercase alphanumeric random suffix
func (s *volumeSlugService) generateRandomSuffix() string {
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
