package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewProjectUserRole(t *testing.T) {
	t.Run("유효한 역할", func(t *testing.T) {
		role, err := NewProjectUserRole("owner")
		require.NoError(t, err)
		assert.Equal(t, ProjectUserRoleOwner, role)
	})

	t.Run("잘못된 역할", func(t *testing.T) {
		invalidRoles := []string{
			"member",
			"guest",
			"admin",
			"invalid",
			"",
		}

		for _, invalid := range invalidRoles {
			t.Run(invalid, func(t *testing.T) {
				role, err := NewProjectUserRole(invalid)
				assert.Error(t, err)
				assert.Equal(t, projecterrors.ErrInvalidUserRole, err)
				assert.Empty(t, role)
			})
		}
	})
}

func TestProjectUserRole_IsValid(t *testing.T) {
	assert.True(t, ProjectUserRoleOwner.IsValid())

	invalidRole := ProjectUserRole("invalid")
	assert.False(t, invalidRole.IsValid())
}

func TestProjectUserRole_IsOwner(t *testing.T) {
	assert.True(t, ProjectUserRoleOwner.IsOwner())

	invalidRole := ProjectUserRole("member")
	assert.False(t, invalidRole.IsOwner())
}

func TestProjectUserRole_String(t *testing.T) {
	assert.Equal(t, "owner", ProjectUserRoleOwner.String())
}

func TestProjectUserRole_Equals(t *testing.T) {
	role1 := ProjectUserRoleOwner
	role2 := ProjectUserRoleOwner
	role3 := ProjectUserRole("other")

	assert.True(t, role1.Equals(role2))
	assert.False(t, role1.Equals(role3))
}
