package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func defaultLimits() value.ResourceLimits {
	limits, _ := value.NewResourceLimits(100, 512, 1024, 1000)
	return *limits
}

func TestGetProjectUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: ID로 프로젝트 조회", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetProjectUseCase(mockProjectService, mockVolumeService)

		projectID := uint(1)
		input := GetProjectInput{
			ProjectID: projectID,
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProjectWithVolumes(projectID, "테스트 프로젝트", *slug, 1)
		volumes := []*volumemodel.Volume{
			createTestVolume(1, projectID, "test-volume", 1024),
		}

		mockProjectService.On("GetProject", ctx, projectID).Return(project, nil)
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return(volumes, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, projectID, output.ProjectID)
		assert.Equal(t, "테스트 프로젝트", output.Name)
		assert.Equal(t, "p2025011812000012345678", output.Slug)
		assert.Equal(t, "active", output.Status)
		assert.Len(t, output.Users, 1)
		assert.Equal(t, uint(1), output.Users[0].UserID)
		assert.Equal(t, "owner", output.Users[0].Role)
		assert.Len(t, output.Volumes, 1)
		assert.Equal(t, uint(1), output.Volumes[0].VolumeID)
		assert.Equal(t, "test-volume", output.Volumes[0].Name)
		assert.Equal(t, uint32(1024), output.Volumes[0].Capacity)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 볼륨이 없는 프로젝트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetProjectUseCase(mockProjectService, mockVolumeService)

		projectID := uint(1)
		input := GetProjectInput{
			ProjectID: projectID,
		}

		slug, _ := value.NewProjectSlug("p2025011812000011111111")
		project := createTestProjectWithVolumes(projectID, "빈 프로젝트", *slug, 1)

		mockProjectService.On("GetProject", ctx, projectID).Return(project, nil)
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*volumemodel.Volume{}, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Volumes, 0)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 볼륨 조회 실패", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetProjectUseCase(mockProjectService, mockVolumeService)

		projectID := uint(1)
		input := GetProjectInput{
			ProjectID: projectID,
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProjectWithVolumes(projectID, "테스트 프로젝트", *slug, 1)

		mockProjectService.On("GetProject", ctx, projectID).Return(project, nil)
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*volumemodel.Volume{}, errors.New("volume service error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "volume service error")
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: ID로 프로젝트 조회 실패", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetProjectUseCase(mockProjectService, mockVolumeService)

		projectID := uint(999)
		input := GetProjectInput{
			ProjectID: projectID,
		}

		mockProjectService.On("GetProject", ctx, projectID).Return((*projectmodel.Project)(nil), projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertNotCalled(t, "ListVolumesByProjectID")
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetProjectUseCase(mockProjectService, mockVolumeService)

		projectID := uint(1)
		input := GetProjectInput{
			ProjectID: projectID,
		}

		mockProjectService.On("GetProject", ctx, projectID).Return((*projectmodel.Project)(nil), errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertNotCalled(t, "ListVolumesByProjectID")
	})
}

func createTestProjectWithVolumes(id uint, name string, slug value.ProjectSlug, ownerID uint) *projectmodel.Project {
	project, _ := projectmodel.NewProject(name, slug, ownerID, defaultLimits(), nil, nil)
	project.SetProjectID(id)

	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	diskLimit := uint32(2048)
	trafficLimit := uint32(128)

	limits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
	_ = project.SetResourceLimits(*limits)

	return project
}

func createTestVolume(id uint, projectID uint, name string, capacity uint32) *volumemodel.Volume {
	volume, _ := volumemodel.NewVolume(projectID, name, capacity)
	volume.SetVolumeID(id)
	return volume
}
