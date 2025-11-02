package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

func defaultLimits() value.ResourceLimits {
	limits, _ := value.NewResourceLimits(500, 512, 1024, 1000) // MinCPULimit: 500
	return *limits
}

func TestNewProject(t *testing.T) {
	t.Run("성공: 유효한 프로젝트 생성", func(t *testing.T) {
		name := "My Project"
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		ownerID := uint(100)

		project, err := NewProject(name, *slug, ownerID, defaultLimits(), nil)

		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, name, project.Name())
		assert.Equal(t, *slug, project.Slug())
		assert.Equal(t, value.ProjectStatusActive, project.Status())
		assert.Equal(t, value.ProjectOperationStatusNothing, project.OperationStatus())
		assert.False(t, project.IsDeleted())
		assert.NotZero(t, project.CreatedAt())
		assert.NotZero(t, project.UpdatedAt())

		// 초기 owner 확인
		users := project.Users()
		assert.Len(t, users, 1)
		assert.Equal(t, ownerID, users[0].UserID())
		assert.True(t, users[0].IsOwner())
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, err := NewProject("", *slug, 100, defaultLimits(), nil)

		assert.Error(t, err)
		assert.Nil(t, project)
	})

	t.Run("실패: 잘못된 owner ID", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, err := NewProject("My Project", *slug, 0, defaultLimits(), nil)

		assert.Error(t, err)
		assert.Nil(t, project)
	})
}

func TestProject_SetProjectID(t *testing.T) {
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, err := NewProject("My Project", *slug, 100, defaultLimits(), nil)
	require.NoError(t, err)
	require.NotNil(t, project)

	project.SetProjectID(999)

	assert.Equal(t, uint(999), project.ProjectID())

	// 사용자들의 projectID도 업데이트되는지 확인
	users := project.Users()
	for _, user := range users {
		assert.Equal(t, uint(999), user.ProjectID())
	}
}

func TestProject_AddUser(t *testing.T) {
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
	project.SetProjectID(1) // Important: Set project ID so users have correct projectID

	t.Run("성공: Owner 추가", func(t *testing.T) {
		err := project.AddUser(101, value.ProjectUserRoleOwner)

		require.NoError(t, err)
		users := project.Users()
		assert.Len(t, users, 2)
		assert.True(t, project.HasUser(101))
	})

	t.Run("실패: 이미 존재하는 사용자", func(t *testing.T) {
		err := project.AddUser(101, value.ProjectUserRoleOwner)

		assert.Error(t, err)
	})

	t.Run("성공: 삭제된 사용자 복구 (Restore)", func(t *testing.T) {
		// 먼저 사용자를 Owner로 추가
		err := project.AddUser(102, value.ProjectUserRoleOwner)
		require.NoError(t, err)

		user, _ := project.GetUserByID(102)
		assert.True(t, user.IsOwner())

		// 사용자 삭제
		err = project.RemoveUser(102)
		require.NoError(t, err)
		assert.False(t, project.HasUser(102))

		// 다시 추가 (복구)
		err = project.AddUser(102, value.ProjectUserRoleOwner)

		require.NoError(t, err)
		assert.True(t, project.HasUser(102))

		user, _ = project.GetUserByID(102)
		assert.True(t, user.IsOwner())

		// GetUsers()를 호출하면 복구된 1개만 존재 (새 객체가 생성되지 않음)
		allUsers := project.Users()
		count102 := 0
		for _, u := range allUsers {
			if u.UserID() == 102 {
				count102++
			}
		}
		assert.Equal(t, 1, count102) // 복구되었으므로 1개만 존재 (이전에는 2개였음)
	})

	t.Run("실패: 삭제된 프로젝트에 사용자 추가", func(t *testing.T) {
		_ = project.SoftDelete()
		err := project.AddUser(103, value.ProjectUserRoleOwner)

		assert.Error(t, err)
	})
}

func TestProject_RemoveUser(t *testing.T) {
	t.Run("성공: Member 제거", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)

		err := project.RemoveUser(101)

		require.NoError(t, err)
		assert.False(t, project.HasUser(101))
	})

	t.Run("실패: 마지막 Owner 제거", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.RemoveUser(100)

		assert.Error(t, err)
		assert.True(t, project.HasUser(100))
	})

	t.Run("성공: 여러 Owner 중 하나 제거", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)

		err := project.RemoveUser(100)

		require.NoError(t, err)
		assert.False(t, project.HasUser(100))
		assert.True(t, project.HasOwner())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.RemoveUser(999)

		assert.Error(t, err)
	})

	t.Run("실패: 삭제된 프로젝트에서 사용자 제거", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)
		_ = project.SoftDelete()

		err := project.RemoveUser(101)

		assert.Error(t, err)
	})
}

func TestProject_ChangeUserRole(t *testing.T) {
	t.Run("성공: Member를 Owner로 변경", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)

		err := project.ChangeUserRole(101, value.ProjectUserRoleOwner)

		require.NoError(t, err)
		user, _ := project.GetUserByID(101)
		assert.True(t, user.IsOwner())
	})

	t.Run("성공: Owner를 Member로 변경 (다른 Owner 있음)", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)

		err := project.ChangeUserRole(100, value.ProjectUserRoleOwner)

		require.NoError(t, err)
		user, _ := project.GetUserByID(100)
		assert.True(t, user.IsOwner())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.ChangeUserRole(999, value.ProjectUserRoleOwner)

		assert.Error(t, err)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		_ = project.SoftDelete()

		err := project.ChangeUserRole(100, value.ProjectUserRoleOwner)

		assert.Error(t, err)
	})
}

func TestProject_GetOwners(t *testing.T) {
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
	project.SetProjectID(1)
	_ = project.AddUser(101, value.ProjectUserRoleOwner)
	// All users are owners since we only have Owner role
	_ = project.AddUser(102, value.ProjectUserRoleOwner)
	_ = project.AddUser(103, value.ProjectUserRoleOwner)

	owners := project.GetOwners()

	assert.Len(t, owners, 4) // All users are owners
	for _, owner := range owners {
		assert.True(t, owner.IsOwner())
	}
}

func TestProject_UpdateName(t *testing.T) {
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)

	t.Run("성공: 이름 업데이트", func(t *testing.T) {
		err := project.SetName("New Project Name")

		require.NoError(t, err)
		assert.Equal(t, "New Project Name", project.Name())
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		err := project.SetName("")

		assert.Error(t, err)
		assert.Equal(t, "New Project Name", project.Name())
	})
}

func TestProject_DeleteAndRestore(t *testing.T) {
	t.Run("프로젝트 삭제", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		_ = project.AddUser(101, value.ProjectUserRoleOwner)

		err := project.SoftDelete()

		require.NoError(t, err)
		assert.True(t, project.IsDeleted())
		deletedAt, ok := project.DeletedAt()
		assert.True(t, ok)
		assert.NotZero(t, deletedAt)

		// 모든 사용자도 삭제되었는지 확인
		users := project.Users()
		for _, user := range users {
			assert.True(t, user.IsDeleted())
		}
	})

}

// ValidateInvariants test removed as the method was deleted from project.go

// Volume tests moved to separate VolumeService tests since volumes are now separate aggregates
/*
func TestProject_AddVolume(t *testing.T) {
	t.Run("성공: 볼륨 추가", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.AddVolume("data-volume", 100)

		require.NoError(t, err)
		volumes := project.GetVolumes()
		assert.Len(t, volumes, 1)
		assert.Equal(t, "data-volume", volumes[0].GetName())
		assert.Equal(t, uint32(100), volumes[0].GetCapacity())
	})

	t.Run("실패: 중복된 볼륨 이름", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddVolume("data-volume", 100)

		err := project.AddVolume("data-volume", 200)

		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
		volumes := project.GetVolumes()
		assert.Len(t, volumes, 1)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.SoftDelete()

		err := project.AddVolume("data-volume", 100)

		assert.Error(t, err)
	})
}

func TestProject_RemoveVolume(t *testing.T) {
	t.Run("성공: 볼륨 제거", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddVolume("data-volume", 100)
		_ = project.SetVolumeID("data-volume", 999)

		err := project.RemoveVolume(999)

		require.NoError(t, err)
		assert.Len(t, project.GetVolumes(), 0)
	})

	t.Run("실패: 존재하지 않는 볼륨", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.RemoveVolume(999)

		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
	})

	t.Run("실패: 삭제된 프로젝트", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		_ = project.SoftDelete()

		err := project.RemoveVolume(999)

		assert.Error(t, err)
	})
}

func TestProject_UpdateVolume(t *testing.T) {
	t.Run("성공: 볼륨 업데이트", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
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
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)
		_ = project.AddVolume("volume1", 100)
		_ = project.AddVolume("volume2", 100)
		_ = project.SetVolumeID("volume1", 1)
		_ = project.SetVolumeID("volume2", 2)

		err := project.UpdateVolume(2, "volume1", 200)

		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
	})

	t.Run("실패: 존재하지 않는 볼륨", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		project.SetProjectID(1)

		err := project.UpdateVolume(999, "new-name", 200)

		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
	})
}

func TestProject_GetVolumeByName(t *testing.T) {
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
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
	slug, _ := value.NewProjectSlug("p2025011812000088888888")
	project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
	project.SetProjectID(1)
	// All users are owners since we only have Owner role
	_ = project.AddUser(101, value.ProjectUserRoleOwner)
	_ = project.AddUser(102, value.ProjectUserRoleOwner)
	_ = project.RemoveUser(102) // Soft delete

	activeUsers := project.GetActiveUsers()

	assert.Len(t, activeUsers, 2) // Owner and Member only
	for _, user := range activeUsers {
		assert.True(t, user.IsActive())
	}
}

func TestProject_OperationStatus(t *testing.T) {
	t.Run("성공: 새 프로젝트의 초기 operation status는 nothing", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)

		assert.Equal(t, value.ProjectOperationStatusNothing, project.OperationStatus())
	})

	t.Run("실패: StartDeploy - nothing에서 deploying으로 전환 차단 (standalone deploy 불가)", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)

		err := project.StartDeploy(1)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidStatusTransition, err)
		assert.Equal(t, value.ProjectOperationStatusNothing, project.OperationStatus())
	})

	t.Run("성공: StartDeploy - building에서 deploying으로 전환", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil, nil)
		_ = project.StartBuild()

		err := project.StartDeploy(1)

		require.NoError(t, err)
		assert.Equal(t, value.ProjectOperationStatusDeploying, project.OperationStatus())
	})

	t.Run("실패: StartDeploy - 이미 deploying 상태일 때", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil, nil)
		_ = project.StartBuild()
		_ = project.StartDeploy(1)

		err := project.StartDeploy(2)

		assert.Error(t, err)
		assert.Equal(t, value.ProjectOperationStatusDeploying, project.OperationStatus())
	})

	t.Run("성공: CompleteDeploy - deploying에서 nothing으로 전환", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		deploymentID := uint(1)
		_ = project.StartBuild()
		_ = project.StartDeploy(deploymentID)

		err := project.CompleteDeploy(deploymentID)

		require.NoError(t, err)
		assert.Equal(t, value.ProjectOperationStatusNothing, project.OperationStatus())
	})

	t.Run("실패: CompleteDeploy - 다른 deployment가 lock을 소유할 때", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		deploymentID := uint(1)
		_ = project.StartBuild()
		_ = project.StartDeploy(deploymentID)

		err := project.CompleteDeploy(uint(2)) // 다른 deployment ID로 시도

		assert.Error(t, err)
		assert.Equal(t, value.ProjectOperationStatusDeploying, project.OperationStatus())
	})

	t.Run("실패: 삭제된 프로젝트의 operation status 변경", func(t *testing.T) {
		slug, _ := value.NewProjectSlug("p2025011812000088888888")
		project, _ := NewProject("My Project", *slug, 100, defaultLimits(), nil)
		_ = project.SoftDelete()

		err := project.StartDeploy(1)

		assert.Error(t, err)
	})
}
