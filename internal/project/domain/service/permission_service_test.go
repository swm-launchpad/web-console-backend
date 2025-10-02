package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

func TestPermissionService_CanUserModifyProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 소유자가 프로젝트 수정", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(1)

		// Create project with user as owner
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserModifyProject(ctx, userID, projectID)

		require.NoError(t, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: UserID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserModifyProject(ctx, 0, 1)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserModifyProject(ctx, 1, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(999)

		mockProjectRepo.On("FindByID", ctx, projectID).Return((*model.Project)(nil), projecterrors.ErrProjectNotFound)

		err := service.CanUserModifyProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(1)

		mockProjectRepo.On("FindByID", ctx, projectID).Return((*model.Project)(nil), errors.New("database error"))

		err := service.CanUserModifyProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자가 프로젝트에 속하지 않음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(2) // Different user
		projectID := uint(1)

		// Create project with different owner
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1) // Owner is user 1

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserModifyProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자가 소유자가 아님", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		ownerID := uint(1)
		memberID := uint(2)
		projectID := uint(1)

		// Create project with owner (only owner role exists)
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, ownerID)
		// Since only owner role exists, memberID will not be part of the project

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserModifyProject(ctx, memberID, projectID)

		assert.Error(t, err)
		// User is not part of the project at all, so permission denied
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertExpectations(t)
	})
}

func TestPermissionService_CanUserAccessProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 소유자가 프로젝트 접근", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(1)

		// Create project with user as owner
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAccessProject(ctx, userID, projectID)

		require.NoError(t, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 비소유자는 프로젝트에 접근할 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		ownerID := uint(1)
		nonOwnerID := uint(2)
		projectID := uint(1)

		// Create project with owner (only owner role exists)
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, ownerID)
		// Since only owner role exists, nonOwnerID will not be part of the project

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAccessProject(ctx, nonOwnerID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: UserID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserAccessProject(ctx, 0, 1)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserAccessProject(ctx, 1, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(999)

		mockProjectRepo.On("FindByID", ctx, projectID).Return((*model.Project)(nil), projecterrors.ErrProjectNotFound)

		err := service.CanUserAccessProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(1)

		mockProjectRepo.On("FindByID", ctx, projectID).Return((*model.Project)(nil), errors.New("database error"))

		err := service.CanUserAccessProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자가 프로젝트에 속하지 않음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(2) // Different user
		projectID := uint(1)

		// Create project with different owner
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1) // Owner is user 1

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAccessProject(ctx, userID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertExpectations(t)
	})
}

func TestPermissionService_CanUserAddVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 소유자가 볼륨 추가", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		projectID := uint(1)

		// Create project with user as owner
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAddVolume(ctx, userID, projectID)

		require.NoError(t, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 소유자가 아닌 사용자", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		ownerID := uint(1)
		otherUserID := uint(2)
		projectID := uint(1)

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, ownerID)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAddVolume(ctx, otherUserID, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockProjectRepo.AssertExpectations(t)
	})
}

func TestPermissionService_CanUserRemoveVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 소유자가 볼륨 제거", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		volumeID := uint(1)
		projectID := uint(1)

		volume, _ := volumemodel.NewVolume(projectID, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserRemoveVolume(ctx, userID, volumeID)

		require.NoError(t, err)

		mockVolumeRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: VolumeID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserRemoveVolume(ctx, 1, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)

		mockVolumeRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 볼륨을 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		volumeID := uint(999)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return((*volumemodel.Volume)(nil), projecterrors.ErrVolumeNotFound)

		err := service.CanUserRemoveVolume(ctx, 1, volumeID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: 소유자가 아닌 사용자", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		ownerID := uint(1)
		otherUserID := uint(2)
		volumeID := uint(1)
		projectID := uint(1)

		volume, _ := volumemodel.NewVolume(projectID, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, ownerID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserRemoveVolume(ctx, otherUserID, volumeID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockVolumeRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})
}

func TestPermissionService_CanUserAccessVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트 멤버가 볼륨 접근", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		userID := uint(1)
		volumeID := uint(1)
		projectID := uint(1)

		volume, _ := volumemodel.NewVolume(projectID, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAccessVolume(ctx, userID, volumeID)

		require.NoError(t, err)

		mockVolumeRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: VolumeID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		err := service.CanUserAccessVolume(ctx, 1, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)

		mockVolumeRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 볼륨을 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		volumeID := uint(999)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return((*volumemodel.Volume)(nil), projecterrors.ErrVolumeNotFound)

		err := service.CanUserAccessVolume(ctx, 1, volumeID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트에 속하지 않은 사용자", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockVolumeRepo := new(repository.MockVolumeRepository)
		service := NewPermissionService(mockProjectRepo, mockVolumeRepo)

		ownerID := uint(1)
		otherUserID := uint(2)
		volumeID := uint(1)
		projectID := uint(1)

		volume, _ := volumemodel.NewVolume(projectID, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, ownerID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		err := service.CanUserAccessVolume(ctx, otherUserID, volumeID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrPermissionDenied, err)

		mockVolumeRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})
}
