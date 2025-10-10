package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

func TestSlugService_EnsureUniqueSlug(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유일한 Slug", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		slug, _ := value.NewProjectSlug("unique-slug-1234")

		mockProjectRepo.On("ExistsBySlug", ctx, slug.String()).Return(false, nil)

		err := service.EnsureUniqueSlug(ctx, *slug)

		require.NoError(t, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 중복된 Slug", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		slug, _ := value.NewProjectSlug("duplicate-slug")

		mockProjectRepo.On("ExistsBySlug", ctx, slug.String()).Return(true, nil)

		err := service.EnsureUniqueSlug(ctx, *slug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugAlreadyExists, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		slug, _ := value.NewProjectSlug("test-slug")

		mockProjectRepo.On("ExistsBySlug", ctx, slug.String()).Return(false, errors.New("database error"))

		err := service.EnsureUniqueSlug(ctx, *slug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)

		mockProjectRepo.AssertExpectations(t)
	})
}

func TestSlugService_GenerateSlugFromName(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 영문 이름으로 Slug 생성", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "My Awesome Project"

		// Mock successful unique slug check (called exactly once)
		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		assert.Contains(t, slug.String(), "my-awesome-project")
		// Verify format: baseSlug-timestamp-random
		assert.Regexp(t, `^my-awesome-project-\d{14}-\d{4}$`, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 한글 이름으로 Slug 생성", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "내 멋진 프로젝트"

		// Mock successful unique slug check (called exactly once)
		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		// Korean characters should be transliterated by gosimple/slug
		// Verify format: baseSlug-timestamp-random
		assert.Regexp(t, `^.+-\d{14}-\d{4}$`, slug.String())
		assert.LessOrEqual(t, len(slug.String()), 63)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 특수문자가 포함된 이름으로 Slug 생성", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "My Project!@#$%^&*()"

		// Mock successful unique slug check (called exactly once)
		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		assert.Contains(t, slug.String(), "my-project")
		// Special characters may be transliterated, so just verify general format
		assert.Regexp(t, `^.+-\d{14}-\d{4}$`, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 빈 이름으로 Slug 생성 (project 폴백)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := ""

		// Mock successful unique slug check (called exactly once)
		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		// Empty name should fallback to "project"
		assert.Regexp(t, `^project-\d{14}-\d{4}$`, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 생성된 slug가 이미 존재하면 바로 에러 반환", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "Test Project"

		// Slug already exists - no retry, return error immediately
		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(true, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugAlreadyExists, err)
		assert.Empty(t, slug.String())

		// Should call ExistsBySlug exactly once (no retry)
		mockProjectRepo.AssertNumberOfCalls(t, "ExistsBySlug", 1)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "Test Project"

		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, errors.New("database error")).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)
		assert.Empty(t, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 긴 이름은 63자로 제한되며 timestamp와 random은 보존", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		// Very long name that would exceed 63 character limit
		name := "This is a very very very very very very very very very very long project name that should be truncated"

		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		assert.LessOrEqual(t, len(slug.String()), 63)
		// Verify format is preserved: baseSlug-timestamp-random
		assert.Regexp(t, `^.+-\d{14}-\d{4}$`, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 공백만 있는 이름 (project 폴백)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "   "

		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())
		// Empty/whitespace name should fallback to "project"
		assert.Regexp(t, `^project-\d{14}-\d{4}$`, slug.String())

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: slug 형식 검증 - baseSlug-timestamp-random", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "Test Project"

		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Once()

		slug, err := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err)
		assert.NotEmpty(t, slug.String())

		// Verify format: baseSlug-timestamp-random
		parts := strings.Split(slug.String(), "-")
		assert.GreaterOrEqual(t, len(parts), 3, "slug should have at least 3 parts: baseSlug-timestamp-random")

		// Last part should be 4-digit random
		randomPart := parts[len(parts)-1]
		assert.Len(t, randomPart, 4, "random part should be 4 digits")
		assert.Regexp(t, `^\d{4}$`, randomPart)

		// Second to last part should be 14-digit timestamp
		timestampPart := parts[len(parts)-2]
		assert.Len(t, timestampPart, 14, "timestamp part should be 14 digits")
		assert.Regexp(t, `^\d{14}$`, timestampPart)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 여러 번 생성해도 timestamp와 random이 달라서 유니크함", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewSlugService(mockProjectRepo)

		name := "Same Name"

		mockProjectRepo.On("ExistsBySlug", ctx, mock.AnythingOfType("string")).Return(false, nil).Times(3)

		// Generate multiple slugs with same name
		slug1, err1 := service.GenerateSlugFromName(ctx, name)
		slug2, err2 := service.GenerateSlugFromName(ctx, name)
		slug3, err3 := service.GenerateSlugFromName(ctx, name)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)

		// All should be different due to timestamp and random
		assert.NotEqual(t, slug1.String(), slug2.String())
		assert.NotEqual(t, slug2.String(), slug3.String())
		assert.NotEqual(t, slug1.String(), slug3.String())

		mockProjectRepo.AssertExpectations(t)
	})
}
