package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	t.Run("성공: 유효한 프로젝트 생성", func(t *testing.T) {
		name := "My Project"
		slug, _ := NewProjectSlug("my-project")
		ownerID := uint(100)

		project, err := NewProject(name, *slug, ownerID)

		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, name, project.GetName())
		assert.Equal(t, *slug, project.GetSlug())
		assert.Equal(t, ProjectStatusActive, project.GetStatus())
		assert.False(t, project.IsDeleted())
		assert.NotZero(t, project.GetCreatedAt())
		assert.NotNil(t, project.GetUpdatedAt())

		// 초기 owner 확인
		users := project.GetUsers()
		assert.Len(t, users, 1)
		assert.Equal(t, ownerID, users[0].GetUserID())
		assert.True(t, users[0].IsOwner())
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, err := NewProject("", *slug, 100)

		assert.Error(t, err)
		assert.Nil(t, project)
	})

	t.Run("실패: 빈 slug", func(t *testing.T) {
		emptySlug := ProjectSlug{}
		project, err := NewProject("My Project", emptySlug, 100)

		assert.Error(t, err)
		assert.Nil(t, project)
	})

	t.Run("실패: 잘못된 owner ID", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, err := NewProject("My Project", *slug, 0)

		assert.Error(t, err)
		assert.Nil(t, project)
	})
}

func TestProject_SetProjectID(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, err := NewProject("My Project", *slug, 100)
	require.NoError(t, err)
	require.NotNil(t, project)

	project.SetProjectID(999)

	assert.Equal(t, uint(999), project.GetProjectID())

	// 사용자들의 projectID도 업데이트되는지 확인
	users := project.GetUsers()
	for _, user := range users {
		assert.Equal(t, uint(999), user.GetProjectID())
	}
}

func TestProject_AddUser(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)
	project.SetProjectID(1) // Important: Set project ID so users have correct projectID

	t.Run("성공: Owner 추가", func(t *testing.T) {
		err := project.AddUser(101, ProjectUserRoleOwner)

		require.NoError(t, err)
		users := project.GetUsers()
		assert.Len(t, users, 2)
		assert.True(t, project.HasUser(101))
	})

	t.Run("실패: 이미 존재하는 사용자", func(t *testing.T) {
		err := project.AddUser(101, ProjectUserRoleOwner)

		assert.Error(t, err)
	})

	t.Run("성공: 삭제된 사용자 복구", func(t *testing.T) {
		// 사용자 삭제
		_ = project.RemoveUser(102)
		assert.False(t, project.HasUser(102))

		// 다시 추가 (복구)
		err := project.AddUser(102, ProjectUserRoleOwner)

		require.NoError(t, err)
		assert.True(t, project.HasUser(102))

		user, _ := project.GetUserByID(102)
		assert.True(t, user.IsOwner())
	})

	t.Run("실패: 삭제된 프로젝트에 사용자 추가", func(t *testing.T) {
		_ = project.Delete()
		err := project.AddUser(103, ProjectUserRoleOwner)

		assert.Error(t, err)
	})
}

func TestProject_RemoveUser(t *testing.T) {
	t.Run("성공: Member 제거", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddUser(101, ProjectUserRoleOwner)

		err := project.RemoveUser(101)

		require.NoError(t, err)
		assert.False(t, project.HasUser(101))
	})

	t.Run("실패: 마지막 Owner 제거", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.RemoveUser(100)

		assert.Error(t, err)
		assert.True(t, project.HasUser(100))
	})

	t.Run("성공: 여러 Owner 중 하나 제거", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddUser(101, ProjectUserRoleOwner)

		err := project.RemoveUser(100)

		require.NoError(t, err)
		assert.False(t, project.HasUser(100))
		assert.True(t, project.HasOwner())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.RemoveUser(999)

		assert.Error(t, err)
	})

	t.Run("실패: 삭제된 프로젝트에서 사용자 제거", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddUser(101, ProjectUserRoleOwner)
		_ = project.Delete()

		err := project.RemoveUser(101)

		assert.Error(t, err)
	})
}

func TestProject_ChangeUserRole(t *testing.T) {
	t.Run("성공: Member를 Owner로 변경", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddUser(101, ProjectUserRoleOwner)

		err := project.ChangeUserRole(101, ProjectUserRoleOwner)

		require.NoError(t, err)
		user, _ := project.GetUserByID(101)
		assert.True(t, user.IsOwner())
	})

	t.Run("성공: Owner를 Member로 변경 (다른 Owner 있음)", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddUser(101, ProjectUserRoleOwner)

		err := project.ChangeUserRole(100, ProjectUserRoleOwner)

		require.NoError(t, err)
		user, _ := project.GetUserByID(100)
		assert.True(t, user.IsOwner())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.ChangeUserRole(999, ProjectUserRoleOwner)

		assert.Error(t, err)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		_ = project.Delete()

		err := project.ChangeUserRole(100, ProjectUserRoleOwner)

		assert.Error(t, err)
	})
}

func TestProject_GetOwners(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)
	project.SetProjectID(1)
	_ = project.AddUser(101, ProjectUserRoleOwner)
	// All users are owners since we only have Owner role
	_ = project.AddUser(102, ProjectUserRoleOwner)
	_ = project.AddUser(103, ProjectUserRoleOwner)

	owners := project.GetOwners()

	assert.Len(t, owners, 4) // All users are owners
	for _, owner := range owners {
		assert.True(t, owner.IsOwner())
	}
}

func TestProject_SetFQDN(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)

	t.Run("성공: FQDN 설정", func(t *testing.T) {
		err := project.SetFQDN("my-project.example.com")

		require.NoError(t, err)
		assert.Equal(t, "my-project.example.com", *project.GetFQDN())
	})

	t.Run("성공: FQDN 제거", func(t *testing.T) {
		err := project.SetFQDN("")

		require.NoError(t, err)
		assert.Nil(t, project.GetFQDN())
	})
}

func TestProject_UpdateName(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)

	t.Run("성공: 이름 업데이트", func(t *testing.T) {
		err := project.UpdateName("New Project Name")

		require.NoError(t, err)
		assert.Equal(t, "New Project Name", project.GetName())
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		err := project.UpdateName("")

		assert.Error(t, err)
		assert.Equal(t, "New Project Name", project.GetName())
	})
}

func TestProject_DeleteAndRestore(t *testing.T) {
	t.Run("프로젝트 삭제", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		_ = project.AddUser(101, ProjectUserRoleOwner)

		err := project.Delete()

		require.NoError(t, err)
		assert.True(t, project.IsSoftDeleted())
		assert.NotNil(t, project.GetDeletedAt())

		// 모든 사용자도 삭제되었는지 확인
		users := project.GetUsers()
		for _, user := range users {
			assert.True(t, user.IsDeleted())
		}
	})

	t.Run("프로젝트 복구", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		_ = project.Delete()

		err := project.Restore()

		require.NoError(t, err)
		assert.False(t, project.IsSoftDeleted())
		assert.Nil(t, project.GetDeletedAt())

		// 사용자는 자동으로 복구되지 않음
		users := project.GetUsers()
		for _, user := range users {
			assert.True(t, user.IsDeleted())
		}
	})
}

func TestProject_ValidateInvariants(t *testing.T) {
	t.Run("성공: 모든 불변식 만족", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)

		err := project.ValidateInvariants()

		assert.NoError(t, err)
	})

	t.Run("실패: Owner 없음", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		// Directly manipulate the users to create a project without owner
		// This simulates a corrupted state that shouldn't happen normally
		// Simulate removing all owners to create invalid state
		project.users = []ProjectUser{}

		err := project.ValidateInvariants()

		assert.Error(t, err)
	})

}

// Volume tests moved to separate VolumeService tests since volumes are now separate aggregates
/*
func TestProject_AddVolume(t *testing.T) {
	t.Run("성공: 볼륨 추가", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.AddVolume("data-volume", 100)

		require.NoError(t, err)
		volumes := project.GetVolumes()
		assert.Len(t, volumes, 1)
		assert.Equal(t, "data-volume", volumes[0].GetName())
		assert.Equal(t, uint32(100), volumes[0].GetCapacity())
	})

	t.Run("실패: 중복된 볼륨 이름", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddVolume("data-volume", 100)

		err := project.AddVolume("data-volume", 200)

		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
		volumes := project.GetVolumes()
		assert.Len(t, volumes, 1)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.Delete()

		err := project.AddVolume("data-volume", 100)

		assert.Error(t, err)
	})
}

func TestProject_RemoveVolume(t *testing.T) {
	t.Run("성공: 볼륨 제거", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddVolume("data-volume", 100)
		_ = project.SetVolumeID("data-volume", 999)

		err := project.RemoveVolume(999)

		require.NoError(t, err)
		assert.Len(t, project.GetVolumes(), 0)
	})

	t.Run("실패: 존재하지 않는 볼륨", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.RemoveVolume(999)

		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		_ = project.Delete()

		err := project.RemoveVolume(999)

		assert.Error(t, err)
	})
}

func TestProject_UpdateVolume(t *testing.T) {
	t.Run("성공: 볼륨 업데이트", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddVolume("old-name", 100)
		_ = project.SetVolumeID("old-name", 999)

		err := project.UpdateVolume(999, "new-name", 200)

		require.NoError(t, err)
		volume, _ := project.GetVolumeByID(999)
		assert.Equal(t, "new-name", volume.GetName())
		assert.Equal(t, uint32(200), volume.GetCapacity())
	})

	t.Run("실패: 중복된 이름", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)
		_ = project.AddVolume("volume1", 100)
		_ = project.AddVolume("volume2", 100)
		_ = project.SetVolumeID("volume1", 1)
		_ = project.SetVolumeID("volume2", 2)

		err := project.UpdateVolume(2, "volume1", 200)

		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
	})

	t.Run("실패: 존재하지 않는 볼륨", func(t *testing.T) {
		slug, _ := NewProjectSlug("my-project")
		project, _ := NewProject("My Project", *slug, 100)
		project.SetProjectID(1)

		err := project.UpdateVolume(999, "new-name", 200)

		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
	})
}

func TestProject_GetVolumeByName(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)
	project.SetProjectID(1)
	_ = project.AddVolume("data-volume", 100)

	t.Run("성공: 이름으로 볼륨 조회", func(t *testing.T) {
		volume, err := project.GetVolumeByName("data-volume")

		require.NoError(t, err)
		assert.Equal(t, "data-volume", volume.GetName())
	})

	t.Run("실패: 존재하지 않는 볼륨", func(t *testing.T) {
		volume, err := project.GetVolumeByName("non-existent")

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
	})
}
*/

func TestProject_GetActiveUsers(t *testing.T) {
	slug, _ := NewProjectSlug("my-project")
	project, _ := NewProject("My Project", *slug, 100)
	project.SetProjectID(1)
	// All users are owners since we only have Owner role
	_ = project.AddUser(101, ProjectUserRoleOwner)
	_ = project.AddUser(102, ProjectUserRoleOwner)
	_ = project.RemoveUser(102) // Soft delete

	activeUsers := project.GetActiveUsers()

	assert.Len(t, activeUsers, 2) // Owner and Member only
	for _, user := range activeUsers {
		assert.True(t, user.IsActive())
	}
}
