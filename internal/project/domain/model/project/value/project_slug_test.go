package value

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewProjectSlug(t *testing.T) {
	t.Run("성공 케이스", func(t *testing.T) {
		validSlugs := []string{
			"my-project",
			"project123",
			"test-app",
			"web-app-123",
			"a-b",
			"mytest",
			"admin", // No longer reserved
			"api",   // No longer reserved
		}

		for _, valid := range validSlugs {
			t.Run(valid, func(t *testing.T) {
				slug, err := NewProjectSlug(valid)
				require.NoError(t, err)
				assert.NotNil(t, slug)
				assert.Equal(t, valid, slug.String())
			})
		}
	})

	t.Run("실패: 빈 slug", func(t *testing.T) {
		slug, err := NewProjectSlug("")
		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugRequired, err)
		assert.Nil(t, slug)
	})

	t.Run("실패: 공백만 있는 slug", func(t *testing.T) {
		slug, err := NewProjectSlug("   ")
		assert.Error(t, err)
		assert.Nil(t, slug)
	})

	t.Run("실패: 너무 짧은 slug", func(t *testing.T) {
		shortSlugs := []string{"a", "ab"}

		for _, short := range shortSlugs {
			t.Run(short, func(t *testing.T) {
				slug, err := NewProjectSlug(short)
				assert.Error(t, err)
				assert.Equal(t, projecterrors.ErrSlugTooShort, err)
				assert.Nil(t, slug)
			})
		}
	})

	t.Run("실패: 너무 긴 slug", func(t *testing.T) {
		longSlug := strings.Repeat("a", 64)
		slug, err := NewProjectSlug(longSlug)
		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugTooLong, err)
		assert.Nil(t, slug)
	})

	t.Run("실패: 잘못된 형식", func(t *testing.T) {
		invalidFormats := []string{
			"project_name",  // 언더스코어
			"project.name",  // 점
			"project name",  // 공백
			"123project",    // 숫자로 시작
			"-project",      // 하이픈으로 시작
			"project-",      // 하이픈으로 끝
			"project--name", // 연속된 하이픈
			"project@name",  // 특수문자
			"프로젝트",          // 한글
		}

		for _, invalid := range invalidFormats {
			t.Run(invalid, func(t *testing.T) {
				slug, err := NewProjectSlug(invalid)
				assert.Error(t, err)
				assert.Nil(t, slug)
			})
		}
	})

	t.Run("대소문자 변환", func(t *testing.T) {
		slug, err := NewProjectSlug("MY-PROJECT")
		require.NoError(t, err)
		assert.Equal(t, "my-project", slug.String())
	})

	t.Run("앞뒤 공백 제거", func(t *testing.T) {
		slug, err := NewProjectSlug("  my-project  ")
		require.NoError(t, err)
		assert.Equal(t, "my-project", slug.String())
	})
}

func TestProjectSlug_String(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	assert.Equal(t, "my-project", slug.String())
}

func TestProjectSlug_Equals(t *testing.T) {
	slug1, _ := NewProjectSlug("my-project")
	slug2, _ := NewProjectSlug("my-project")
	slug3, _ := NewProjectSlug("other-project")

	assert.True(t, slug1.Equals(*slug2))
	assert.True(t, slug2.Equals(*slug1))
	assert.False(t, slug1.Equals(*slug3))
	assert.False(t, slug3.Equals(*slug1))
}
