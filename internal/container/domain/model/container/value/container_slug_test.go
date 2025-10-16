package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewContainerSlug_ValidSlugs(t *testing.T) {
	validSlugs := []string{
		"backend",
		"frontend-app",
		"api-server",
		"db-mysql",
		"backend-20240101120000-1234",
		"a1b", // minimum length (3)
		"a123456789012345678901234567890123456789012345678901234567890b", // max length (63)
	}

	for _, slug := range validSlugs {
		t.Run(slug, func(t *testing.T) {
			result, err := NewContainerSlug(slug)
			assert.NoError(t, err)
			assert.Equal(t, slug, result.String())
		})
	}
}

func TestNewContainerSlug_TooShort(t *testing.T) {
	_, err := NewContainerSlug("ab")
	assert.ErrorIs(t, err, containererrors.ErrSlugTooShort)
}

func TestNewContainerSlug_TooLong(t *testing.T) {
	longSlug := string(make([]byte, 64)) // 64 characters
	_, err := NewContainerSlug(longSlug)
	assert.ErrorIs(t, err, containererrors.ErrSlugTooLong)
}

func TestNewContainerSlug_InvalidFormat(t *testing.T) {
	invalidSlugs := []string{
		"Backend",   // uppercase
		"back_end",  // underscore
		"-backend",  // starts with hyphen
		"backend-",  // ends with hyphen
		"back--end", // consecutive hyphens
		"back end",  // space
		"backend!",  // special character
		"back.end",  // dot
	}

	for _, slug := range invalidSlugs {
		t.Run(slug, func(t *testing.T) {
			_, err := NewContainerSlug(slug)
			assert.ErrorIs(t, err, containererrors.ErrSlugInvalidFormat)
		})
	}
}

func TestNewContainerSlug_Reserved(t *testing.T) {
	reservedSlugs := []string{
		"api",
		"admin",
		"www",
		"localhost",
		"system",
	}

	for _, slug := range reservedSlugs {
		t.Run(slug, func(t *testing.T) {
			_, err := NewContainerSlug(slug)
			assert.ErrorIs(t, err, containererrors.ErrSlugReserved)
		})
	}
}

func TestContainerSlug_Equals(t *testing.T) {
	slug1, _ := NewContainerSlug("backend")
	slug2, _ := NewContainerSlug("backend")
	slug3, _ := NewContainerSlug("frontend")

	assert.True(t, slug1.Equals(slug2))
	assert.False(t, slug1.Equals(slug3))
}
