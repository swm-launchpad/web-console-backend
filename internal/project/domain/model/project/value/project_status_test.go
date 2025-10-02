package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewProjectStatus(t *testing.T) {
	t.Run("유효한 상태", func(t *testing.T) {
		status, err := NewProjectStatus("active")
		assert.NoError(t, err)
		assert.Equal(t, ProjectStatusActive, status)
	})

	t.Run("잘못된 상태", func(t *testing.T) {
		invalidStatuses := []string{
			"inactive",
			"suspended",
			"pending",
			"unknown",
			"",
		}

		for _, invalid := range invalidStatuses {
			t.Run(invalid, func(t *testing.T) {
				status, err := NewProjectStatus(invalid)
				assert.Error(t, err)
				assert.Equal(t, projecterrors.ErrInvalidFormat, err)
				assert.Empty(t, status)
			})
		}
	})
}

func TestProjectStatus_IsValid(t *testing.T) {
	assert.True(t, ProjectStatusActive.isValid())

	invalidStatus := ProjectStatus("invalid")
	assert.False(t, invalidStatus.isValid())
}

func TestProjectStatus_IsActive(t *testing.T) {
	assert.True(t, ProjectStatusActive.IsActive())

	invalidStatus := ProjectStatus("invalid")
	assert.False(t, invalidStatus.IsActive())
}

func TestProjectStatus_String(t *testing.T) {
	assert.Equal(t, "active", ProjectStatusActive.String())
}

func TestProjectStatus_Equals(t *testing.T) {
	status1 := ProjectStatusActive
	status2 := ProjectStatusActive
	status3 := ProjectStatus("other")

	assert.True(t, status1.Equals(status2))
	assert.False(t, status1.Equals(status3))
}
