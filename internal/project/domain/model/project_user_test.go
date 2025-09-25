package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectUser(t *testing.T) {
	t.Run("성공: 유효한 프로젝트 사용자 생성", func(t *testing.T) {
		projectID := uint(1)
		userID := uint(100)
		role := ProjectUserRoleOwner

		pu, err := NewProjectUser(projectID, userID, role)

		require.NoError(t, err)
		assert.NotNil(t, pu)
		assert.Equal(t, projectID, pu.GetProjectID())
		assert.Equal(t, userID, pu.GetUserID())
		assert.Equal(t, role, pu.GetRole())
		assert.False(t, pu.IsDeleted())
		assert.NotZero(t, pu.GetCreatedAt())
		assert.NotNil(t, pu.GetUpdatedAt())
	})

	t.Run("실패: 잘못된 프로젝트 ID", func(t *testing.T) {
		pu, err := NewProjectUser(0, 100, ProjectUserRoleOwner)

		assert.Error(t, err)
		assert.Nil(t, pu)
	})

	t.Run("실패: 잘못된 사용자 ID", func(t *testing.T) {
		pu, err := NewProjectUser(1, 0, ProjectUserRoleOwner)

		assert.Error(t, err)
		assert.Nil(t, pu)
	})

	t.Run("실패: 잘못된 역할", func(t *testing.T) {
		pu, err := NewProjectUser(1, 100, ProjectUserRole("invalid"))

		assert.Error(t, err)
		assert.Nil(t, pu)
	})
}

func TestProjectUser_ChangeRole(t *testing.T) {
	t.Run("성공: Owner에서 Member로 변경", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		// Test changing to the same role since we only have Owner
		// This is now a no-op test
		originalUpdatedAt := pu.GetUpdatedAt()

		err := pu.ChangeRole(ProjectUserRoleOwner)

		require.NoError(t, err)
		assert.Equal(t, ProjectUserRoleOwner, pu.GetRole())
		// No change means updated time shouldn't change
		assert.Equal(t, originalUpdatedAt, pu.GetUpdatedAt())
	})

	t.Run("성공: 동일한 역할로 변경 (변경 없음)", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		originalUpdatedAt := pu.GetUpdatedAt()

		err := pu.ChangeRole(ProjectUserRoleOwner)

		require.NoError(t, err)
		assert.Equal(t, ProjectUserRoleOwner, pu.GetRole())
		assert.Equal(t, originalUpdatedAt, pu.GetUpdatedAt())
	})

	t.Run("실패: 잘못된 역할", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		err := pu.ChangeRole(ProjectUserRole("invalid"))

		assert.Error(t, err)
		assert.Equal(t, ProjectUserRoleOwner, pu.GetRole())
	})

	t.Run("실패: 삭제된 사용자의 역할 변경", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		_ = pu.SoftDelete()

		err := pu.ChangeRole(ProjectUserRoleOwner)

		assert.Error(t, err)
		assert.Equal(t, ProjectUserRoleOwner, pu.GetRole())
	})
}

func TestProjectUser_RoleCheckers(t *testing.T) {
	t.Run("IsOwner", func(t *testing.T) {
		owner, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		assert.True(t, owner.IsOwner())

		// 삭제된 owner
		_ = owner.SoftDelete()
		assert.False(t, owner.IsOwner())
	})

}

func TestProjectUser_SoftDelete(t *testing.T) {
	t.Run("성공: 소프트 삭제", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		err := pu.SoftDelete()

		require.NoError(t, err)
		assert.True(t, pu.IsDeleted())
		assert.NotNil(t, pu.GetDeletedAt())
		assert.False(t, pu.IsActive())
	})

	t.Run("성공: 이미 삭제된 사용자 (중복 삭제)", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		_ = pu.SoftDelete()
		firstDeletedAt := pu.GetDeletedAt()

		err := pu.SoftDelete()

		require.NoError(t, err)
		assert.True(t, pu.IsDeleted())
		assert.Equal(t, firstDeletedAt, pu.GetDeletedAt())
	})
}

func TestProjectUser_Restore(t *testing.T) {
	t.Run("성공: 삭제된 사용자 복구", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		_ = pu.SoftDelete()

		err := pu.Restore()

		require.NoError(t, err)
		assert.False(t, pu.IsDeleted())
		assert.Nil(t, pu.GetDeletedAt())
		assert.True(t, pu.IsActive())
	})

	t.Run("성공: 삭제되지 않은 사용자 복구 시도", func(t *testing.T) {
		pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		err := pu.Restore()

		require.NoError(t, err)
		assert.False(t, pu.IsDeleted())
	})
}

func TestProjectUser_Equals(t *testing.T) {
	t.Run("동일한 ProjectUser", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		assert.True(t, pu1.Equals(pu2))
	})

	t.Run("다른 프로젝트 ID", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2, _ := NewProjectUser(2, 100, ProjectUserRoleOwner)

		assert.False(t, pu1.Equals(pu2))
	})

	t.Run("다른 사용자 ID", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2, _ := NewProjectUser(1, 101, ProjectUserRoleOwner)

		assert.False(t, pu1.Equals(pu2))
	})

	t.Run("다른 역할", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2.role = ProjectUserRole("other") // Force different role for testing

		assert.False(t, pu1.Equals(pu2))
	})

	t.Run("다른 삭제 상태", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		pu2, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)
		_ = pu2.SoftDelete()

		assert.False(t, pu1.Equals(pu2))
	})

	t.Run("nil 비교", func(t *testing.T) {
		pu1, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

		assert.False(t, pu1.Equals(nil))
	})
}

func TestProjectUser_BelongsToProject(t *testing.T) {
	pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

	assert.True(t, pu.BelongsToProject(1))
	assert.False(t, pu.BelongsToProject(2))
}

func TestProjectUser_BelongsToUser(t *testing.T) {
	pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

	assert.True(t, pu.BelongsToUser(100))
	assert.False(t, pu.BelongsToUser(101))
}

func TestProjectUser_IsActive(t *testing.T) {
	pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

	assert.True(t, pu.IsActive())

	_ = pu.SoftDelete()
	assert.False(t, pu.IsActive())

	_ = pu.Restore()
	assert.True(t, pu.IsActive())
}

func TestProjectUser_SetProjectUserID(t *testing.T) {
	pu, _ := NewProjectUser(1, 100, ProjectUserRoleOwner)

	assert.Equal(t, uint(0), pu.GetProjectUserID())

	pu.SetProjectUserID(999)
	assert.Equal(t, uint(999), pu.GetProjectUserID())
}
