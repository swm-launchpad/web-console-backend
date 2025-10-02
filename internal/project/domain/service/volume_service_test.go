package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

func defaultProjectLimits() value.ResourceLimits {
	limits, _ := value.NewResourceLimits(100, 512, 1024, 1000)
	return *limits
}

func TestVolumeService_CreateVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 볼륨 생성", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		name := "test-volume"
		capacity := uint32(1024)

		// Create project with disk limit
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)
		diskLimit := uint32(2048)
		limits, _ := value.NewResourceLimits(100, 128, diskLimit, 128)
		_ = project.SetResourceLimits(*limits)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("ExistsByName", ctx, projectID, name).Return(false, nil)
		mockVolumeRepo.On("GetTotalCapacityByProjectID", ctx, projectID).Return(uint32(0), nil)
		mockVolumeRepo.On("Create", ctx, mock.MatchedBy(func(volume *model.Volume) bool {
			return volume.ProjectID() == projectID && volume.Name() == name && volume.Capacity() == capacity
		})).Return(nil)

		volume, err := service.CreateVolume(ctx, projectID, name, capacity)

		require.NoError(t, err)
		assert.NotNil(t, volume)
		assert.Equal(t, projectID, volume.ProjectID())
		assert.Equal(t, name, volume.Name())
		assert.Equal(t, capacity, volume.Capacity())

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("성공: 기존 볼륨과 함께 새 볼륨 생성", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		name := "new-volume"
		capacity := uint32(512)

		// Create project with disk limit
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)
		diskLimit := uint32(2048)
		limits, _ := value.NewResourceLimits(100, 128, diskLimit, 128)
		_ = project.SetResourceLimits(*limits)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("ExistsByName", ctx, projectID, name).Return(false, nil)
		mockVolumeRepo.On("GetTotalCapacityByProjectID", ctx, projectID).Return(uint32(1024), nil)
		mockVolumeRepo.On("Create", ctx, mock.Anything).Return(nil)

		volume, err := service.CreateVolume(ctx, projectID, name, capacity)

		require.NoError(t, err)
		assert.NotNil(t, volume)

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volume, err := service.CreateVolume(ctx, 0, "test-volume", 1024)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, volume)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
		mockVolumeRepo.AssertNotCalled(t, "ExistsByName")
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(999)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return((*projectmodel.Project)(nil), projecterrors.ErrProjectNotFound)

		volume, err := service.CreateVolume(ctx, projectID, "test-volume", 1024)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, volume)

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertNotCalled(t, "ExistsByName")
	})

	t.Run("실패: 중복된 볼륨 이름", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		name := "duplicate-volume"

		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("ExistsByName", ctx, projectID, name).Return(true, nil)

		volume, err := service.CreateVolume(ctx, projectID, name, 1024)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
		assert.Nil(t, volume)

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertExpectations(t)
		mockVolumeRepo.AssertNotCalled(t, "FindByProjectID")
		mockVolumeRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: 디스크 제한 초과", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		name := "large-volume"
		capacity := uint32(2048)

		// Create project with disk limit
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)
		diskLimit := uint32(2048)
		limits, _ := value.NewResourceLimits(100, 128, diskLimit, 128)
		_ = project.SetResourceLimits(*limits)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("ExistsByName", ctx, projectID, name).Return(false, nil)
		mockVolumeRepo.On("GetTotalCapacityByProjectID", ctx, projectID).Return(uint32(1500), nil)

		volume, err := service.CreateVolume(ctx, projectID, name, capacity)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeDiskLimitExceeded, err)
		assert.Nil(t, volume)

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertExpectations(t)
		mockVolumeRepo.AssertNotCalled(t, "Create")
	})

	t.Run("성공: 명시적 디스크 제한이 있는 경우", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		name := "test-volume"
		capacity := uint32(1024)

		// Create project with explicit disk limit
		slug, _ := value.NewProjectSlug("test-project")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)
		diskLimit := uint32(2048)
		limits, _ := value.NewResourceLimits(100, 128, diskLimit, 128)
		_ = project.SetResourceLimits(*limits)

		mockProjectRepo.On("FindByIDForUpdate", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("ExistsByName", ctx, projectID, name).Return(false, nil)
		mockVolumeRepo.On("GetTotalCapacityByProjectID", ctx, projectID).Return(uint32(0), nil)
		mockVolumeRepo.On("Create", ctx, mock.Anything).Return(nil)

		volume, err := service.CreateVolume(ctx, projectID, name, capacity)

		require.NoError(t, err)
		assert.NotNil(t, volume)

		mockProjectRepo.AssertExpectations(t)
		mockVolumeRepo.AssertExpectations(t)
	})
}

func TestVolumeService_GetVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 ID로 볼륨 조회", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volumeID := uint(1)
		expectedVolume, _ := model.NewVolume(1, "test-volume", 1024)
		expectedVolume.SetVolumeID(volumeID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(expectedVolume, nil)

		volume, err := service.GetVolume(ctx, volumeID)

		require.NoError(t, err)
		assert.NotNil(t, volume)
		assert.Equal(t, expectedVolume, volume)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: VolumeID가 0", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volume, err := service.GetVolume(ctx, 0)

		assert.Error(t, err)
		assert.Nil(t, volume)

		mockVolumeRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 볼륨을 찾을 수 없음", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volumeID := uint(999)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return((*model.Volume)(nil), projecterrors.ErrVolumeNotFound)

		volume, err := service.GetVolume(ctx, volumeID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
		assert.Nil(t, volume)

		mockVolumeRepo.AssertExpectations(t)
	})
}

func TestVolumeService_ListVolumesByProjectID(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트의 볼륨 목록 조회", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)
		volume1, _ := model.NewVolume(projectID, "volume-1", 1024)
		volume2, _ := model.NewVolume(projectID, "volume-2", 2048)
		expectedVolumes := []*model.Volume{volume1, volume2}

		// Project exists check
		slug, _ := value.NewProjectSlug("test-project")
		project, _ := projectmodel.NewProject("Test Project", *slug, 1, defaultProjectLimits(), nil, nil)
		project.SetProjectID(projectID)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockVolumeRepo.On("FindByProjectID", ctx, projectID).Return(expectedVolumes, nil)

		volumes, err := service.ListVolumesByProjectID(ctx, projectID)

		require.NoError(t, err)
		assert.Len(t, volumes, 2)
		assert.Equal(t, expectedVolumes, volumes)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volumes, err := service.ListVolumesByProjectID(ctx, 0)

		assert.Error(t, err)
		assert.Nil(t, volumes)

		mockVolumeRepo.AssertNotCalled(t, "FindByProjectID")
	})
}

func TestVolumeService_DeleteVolume(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 볼륨 삭제", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volumeID := uint(1)

		// Create a test volume
		volume, _ := model.NewVolume(1, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockVolumeRepo.On("Delete", ctx, volumeID).Return(nil)

		err := service.DeleteVolume(ctx, volumeID)

		require.NoError(t, err)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: VolumeID가 0", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		err := service.DeleteVolume(ctx, 0)

		assert.Error(t, err)

		mockVolumeRepo.AssertNotCalled(t, "Delete")
	})

	t.Run("실패: 저장소 삭제 에러", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		volumeID := uint(1)
		projectID := uint(1)

		// Create a volume to be deleted
		volume, _ := model.NewVolume(projectID, "test-volume", 1024)
		volume.SetVolumeID(volumeID)

		mockVolumeRepo.On("FindByID", ctx, volumeID).Return(volume, nil)
		mockVolumeRepo.On("Delete", ctx, volumeID).Return(errors.New("database error"))

		err := service.DeleteVolume(ctx, volumeID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		mockVolumeRepo.AssertExpectations(t)
	})
}

func TestVolumeService_DeleteVolumesByProjectID(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트의 모든 볼륨 삭제", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		projectID := uint(1)

		mockVolumeRepo.On("DeleteByProjectID", ctx, projectID).Return(nil)

		err := service.DeleteVolumesByProjectID(ctx, projectID)

		require.NoError(t, err)

		mockVolumeRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockVolumeRepo := new(repository.MockVolumeRepository)
		mockProjectRepo := new(repository.MockProjectRepository)
		service := NewVolumeService(mockVolumeRepo, mockProjectRepo)

		err := service.DeleteVolumesByProjectID(ctx, 0)

		assert.Error(t, err)

		mockVolumeRepo.AssertNotCalled(t, "DeleteByProjectID")
	})
}
