package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/mock"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

func TestSlugService_GenerateSlug(t *testing.T) {
	t.Run("successful slug generation", func(t *testing.T) {
		mockRepo := new(repository.MockProjectRepository)

		// Mock that slug does not exist
		mockRepo.On("ExistsBySlug", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)

		service := NewSlugService(mockRepo)
		slug, err := service.GenerateSlug(context.Background())

		if err != nil {
			t.Errorf("GenerateSlug() unexpected error: %v", err)
			return
		}

		// Validate slug format
		regex := regexp.MustCompile(`^p[0-9]{14}[a-z0-9]{8}$`)
		if !regex.MatchString(slug.String()) {
			t.Errorf("GenerateSlug() = %v, does not match pattern", slug.String())
		}
		if len(slug.String()) != 23 {
			t.Errorf("GenerateSlug() length = %d, want 23", len(slug.String()))
		}

		mockRepo.AssertExpectations(t)
	})

	t.Run("slug already exists", func(t *testing.T) {
		mockRepo := new(repository.MockProjectRepository)

		// Mock that slug exists
		mockRepo.On("ExistsBySlug", mock.Anything, mock.AnythingOfType("string")).Return(true, nil)

		service := NewSlugService(mockRepo)
		_, err := service.GenerateSlug(context.Background())

		if err != projecterrors.ErrSlugAlreadyExists {
			t.Errorf("GenerateSlug() error = %v, want %v", err, projecterrors.ErrSlugAlreadyExists)
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestSlugService_EnsureUniqueSlug(t *testing.T) {
	t.Run("unique slug", func(t *testing.T) {
		mockRepo := new(repository.MockProjectRepository)
		testSlug := "p2025011812000012345678"

		mockRepo.On("ExistsBySlug", mock.Anything, testSlug).Return(false, nil)

		service := NewSlugService(mockRepo)
		slug, _ := value.NewProjectSlug(testSlug)
		err := service.EnsureUniqueSlug(context.Background(), *slug)

		if err != nil {
			t.Errorf("EnsureUniqueSlug() unexpected error: %v", err)
		}

		mockRepo.AssertExpectations(t)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		mockRepo := new(repository.MockProjectRepository)
		testSlug := "p2025011812000012345678"

		mockRepo.On("ExistsBySlug", mock.Anything, testSlug).Return(true, nil)

		service := NewSlugService(mockRepo)
		slug, _ := value.NewProjectSlug(testSlug)
		err := service.EnsureUniqueSlug(context.Background(), *slug)

		if err != projecterrors.ErrSlugAlreadyExists {
			t.Errorf("EnsureUniqueSlug() error = %v, want %v", err, projecterrors.ErrSlugAlreadyExists)
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestSlugService_generateSlug(t *testing.T) {
	service := &slugService{}

	// Test multiple slug generations
	slugs := make(map[string]bool)
	regex := regexp.MustCompile(`^p[0-9]{14}[a-z0-9]{8}$`)

	for i := 0; i < 100; i++ {
		slug := service.generateSlug()

		// Check length
		if len(slug) != 23 {
			t.Errorf("generateSlug() length = %d, want 23", len(slug))
		}

		// Check format
		if !regex.MatchString(slug) {
			t.Errorf("generateSlug() = %v, does not match pattern", slug)
		}

		// Check uniqueness (note: there's a tiny chance of collision)
		if slugs[slug] {
			t.Logf("Warning: duplicate slug generated: %s (this is statistically rare but possible)", slug)
		}
		slugs[slug] = true
	}
}
