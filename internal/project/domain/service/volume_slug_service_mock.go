package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
)

// MockVolumeSlugService is a mock implementation of VolumeSlugService for testing
type MockVolumeSlugService struct {
	EnsureUniqueSlugFunc func(ctx context.Context, slug value.VolumeSlug) error
	GenerateSlugFunc     func(ctx context.Context) (value.VolumeSlug, error)
}

func (m *MockVolumeSlugService) EnsureUniqueSlug(ctx context.Context, slug value.VolumeSlug) error {
	if m.EnsureUniqueSlugFunc != nil {
		return m.EnsureUniqueSlugFunc(ctx, slug)
	}
	return nil
}

func (m *MockVolumeSlugService) GenerateSlug(ctx context.Context) (value.VolumeSlug, error) {
	if m.GenerateSlugFunc != nil {
		return m.GenerateSlugFunc(ctx)
	}
	// Return a default slug for testing
	return value.NewVolumeSlug("v2025011812000012345678")
}
